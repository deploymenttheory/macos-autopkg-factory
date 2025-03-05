// suspiciousPackageHelpers.js
// JavaScript functions for interacting with Suspicious Package via AppleScript

/**
 * Executes an AppleScript command and returns the result
 * @param {string} script - The AppleScript to execute
 * @returns {Promise<string>} - Result of the AppleScript execution
 */
async function runAppleScript(script) {
  const { exec } = require('child_process');
  
  return new Promise((resolve, reject) => {
    exec(`osascript -e '${script.replace(/'/g, "'\\''")}'`, (error, stdout, stderr) => {
      if (error) {
        reject(new Error(`AppleScript execution error: ${error.message}`));
        return;
      }
      resolve(stdout.trim());
    });
  });
}

/**
 * Opens a package file in Suspicious Package
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<string>} - Result of the open operation
 */
async function openPackage(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
    end tell
  `;
  
  return runAppleScript(script);
}

/**
 * Gets all installed items from a package
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - Array of installed items
 */
async function getInstalledItems(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set itemList to {}
      repeat with anItem in installed items of first document
        set itemList to itemList & {{name:name of anItem, path:posix path of anItem, kind:kind of anItem}}
      end repeat
      return itemList
    end tell
  `;
  
  const result = await runAppleScript(script);
  // Parse the AppleScript result into a JavaScript array
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Gets all installed apps from a package
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - Array of installed apps
 */
async function getInstalledApps(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set appList to (find apps under installed items of first document)
      set resultList to {}
      repeat with anApp in appList
        set appInfo to {name:name of anApp, path:posix path of anApp, bundleID:bundle identifier of anApp}
        set resultList to resultList & {appInfo}
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  // Parse the AppleScript result
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Checks if a package is properly signed
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Object>} - Signature status information
 */
async function checkPackageSignature(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set pkg to first document
      set sigStatus to signature status of installer package of pkg
      set isNotarized to notarized of installer package of pkg
      return {status:sigStatus, notarized:isNotarized}
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '{' : '}';
  }));
}

/**
 * Gets all scripts included in the package
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - Array of installer scripts
 */
async function getInstallerScripts(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set scriptList to {}
      repeat with aScript in installer scripts of first document
        set scriptInfo to {name:name of aScript, when:runs when of aScript, binary:binary of aScript}
        if not (binary of aScript) then
          set scriptInfo to scriptInfo & {text:installer script text of aScript}
        end if
        set scriptList to scriptList & {scriptInfo}
      end repeat
      return scriptList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Finds issues with a package
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - Array of issues found
 */
async function findPackageIssues(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set issueList to {}
      repeat with anIssue in issues of first document
        set issueInfo to {details:details of anIssue, priority:priority of anIssue}
        set relPath to related POSIX path of anIssue
        if relPath is not missing value then
          set issueInfo to issueInfo & {path:relPath}
        end if
        set issueList to issueList & {issueInfo}
      end repeat
      return issueList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Finds all components with specific entitlements
 * @param {string} packagePath - Full path to the package file
 * @param {string} entitlementKey - The entitlement key to search for
 * @returns {Promise<Array>} - Array of components with the entitlement
 */
async function findComponentsWithEntitlement(packagePath, entitlementKey) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set resultList to {}
      set allCode to (find code under installed items of first document)
      repeat with codeItem in allCode
        set entitlementObj to entitlement "${entitlementKey}" of codeItem
        if entitlementObj is not missing value then
          set itemInfo to {name:name of codeItem, path:posix path of codeItem, value:string value of entitlementObj}
          set resultList to resultList & {itemInfo}
        end if
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Exports a diffable manifest from a package
 * @param {string} packagePath - Full path to the package file
 * @param {string} outputPath - Path where to save the manifest
 * @returns {Promise<string>} - Result of the export operation
 */
async function exportDiffableManifest(packagePath, outputPath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      export diffable manifest of installer package of first document to "${outputPath}" with other subfolders preserved
      return "Manifest exported to ${outputPath}"
    end tell
  `;
  
  return runAppleScript(script);
}

/**
 * Checks whether executables in a package support a specific architecture
 * @param {string} packagePath - Full path to the package file
 * @param {string} architecture - Architecture to check for ("arm64", "arm64e", "x86_64", etc.)
 * @returns {Promise<Array>} - List of executables and whether they support the architecture
 */
async function checkArchitectureSupport(packagePath, architecture) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set allCode to (find code under installed items of first document)
      set resultList to {}
      repeat with codeItem in allCode
        set doesSupport to codeItem supports processor "${architecture}"
        set itemInfo to {name:name of codeItem, path:posix path of codeItem, supports:doesSupport}
        set resultList to resultList & {itemInfo}
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

// Export all functions
module.exports = {
  openPackage,
  getInstalledItems,
  getInstalledApps,
  checkPackageSignature,
  getInstallerScripts,
  findPackageIssues,
  findComponentsWithEntitlement,
  exportDiffableManifest,
  checkArchitectureSupport
};

// Additional helper functions for the JavaScript module

/**
 * Finds all launchd job configuration files in a package
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - Array of launchd configuration files with their details
 */
async function findLaunchdJobs(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set launchdJobs to (find content under installed items of first document whose kind is "Launchd job configuration")
      set resultList to {}
      repeat with aJob in launchdJobs
        set jobInfo to {name:name of aJob, path:posix path of aJob, owner:owner of aJob, ownerID:owner ID of aJob, permissions:symbolic permissions of aJob}
        set resultList to resultList & {jobInfo}
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Finds installer scripts with root privileges
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - Array of privileged scripts with their details
 */
async function findPrivilegedScripts(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set rootScripts to installer scripts of first document whose runs as user is "root"
      set resultList to {}
      repeat with aScript in rootScripts
        set scriptInfo to {name:name of aScript, shortName:short name of aScript, when:runs when of aScript, binary:binary of aScript}
        set resultList to resultList & {scriptInfo}
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Gets the most critical issues from a package
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - Array of critical and warning issues
 */
async function findCriticalIssues(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set criticalIssues to (every issue of first document where priority is critical or priority is warning)
      set resultList to {}
      repeat with anIssue in criticalIssues
        set issueInfo to {details:details of anIssue, priority:priority of anIssue}
        set relPath to related POSIX path of anIssue
        if relPath is not missing value then
          set issueInfo to issueInfo & {path:relPath}
        end if
        set resultList to resultList & {issueInfo}
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Gets the OS version requirements for all executable components in a package
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - Array of executables with their OS version requirements
 */
async function getOSRequirements(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set codeItems to (find code under installed items of first document)
      set resultList to {}
      repeat with anItem in codeItems
        set minOS to OS minimum version of anItem
        if minOS is not missing value then
          set itemInfo to {name:name of anItem, path:posix path of anItem}
          set itemInfo to itemInfo & {platform:platform of minOS, version:product version of minOS, major:major version number of minOS}
          set resultList to resultList & {itemInfo}
        end if
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Checks if executables in a package support a specific macOS version
 * @param {string} packagePath - Full path to the package file
 * @param {string} osVersion - OS version to check (e.g. "14.0" for Sonoma)
 * @returns {Promise<Array>} - List of executables that may not be compatible
 */
async function checkOSCompatibility(packagePath, osVersion) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set codeItems to (find code under installed items of first document)
      set resultList to {}
      repeat with anItem in codeItems
        set minOS to OS minimum version of anItem
        if minOS is not missing value then
          set isCompatible to check OS version minOS is before or at "${osVersion}"
          if not isCompatible then
            set itemInfo to {name:name of anItem, path:posix path of anItem, required:product version of minOS}
            set resultList to resultList & {itemInfo}
          end if
        end if
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Checks for files with non-standard permissions
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - List of items with unusual permissions
 */
async function findNonStandardPermissions(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      -- Find items with unusual permissions (non-standard or overly permissive)
      set permItems to (find content under installed items of first document where everyone's privileges is not "read only" or owner is not "root")
      set resultList to {}
      repeat with anItem in permItems
        set itemInfo to {name:name of anItem, path:posix path of anItem, symPerm:symbolic permissions of anItem, owner:owner of anItem, group:group of anItem}
        set resultList to resultList & {itemInfo}
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Searches for specific strings in installer scripts (useful for security auditing)
 * @param {string} packagePath - Full path to the package file
 * @param {string} searchTerm - String to search for in scripts
 * @returns {Promise<Array>} - List of scripts containing the search term
 */
async function searchInstallerScripts(packagePath, searchTerm) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set matchingScripts to installer scripts of first document whose installer script text contains "${searchTerm}"
      set resultList to {}
      repeat with aScript in matchingScripts
        set scriptInfo to {name:name of aScript, shortName:short name of aScript, when:runs when of aScript, user:runs as user of aScript}
        set resultList to resultList & {scriptInfo}
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Gets information about component packages and their installation history
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - List of component packages with install history
 */
async function getComponentPackages(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set allComponents to component packages of first document
      set resultList to {}
      repeat with aComponent in allComponents
        set compInfo to {id:name of aComponent, version:package version of aComponent, installed:installed of aComponent}
        if installed of aComponent then
          set compInfo to compInfo & {installedVersion:installed version of aComponent, installedDate:installed date of aComponent, installingApp:installing app of aComponent}
        end if
        set resultList to resultList & {compInfo}
      end repeat
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Find items that match a specific UTI conformance pattern
 * @param {string} packagePath - Full path to the package file
 * @param {string} utiPattern - UTI pattern to test for (e.g. "com.apple.bundle") 
 * @returns {Promise<Array>} - List of items conforming to the UTI
 */
async function findItemsByUTI(packagePath, utiPattern) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set resultList to {}
      
      -- First make sure the UTI exists in the document
      set docRef to first document
      set utiExists to false
      try
        set testUTI to uniform type ID "${utiPattern}" of docRef
        set utiExists to true
      end try
      
      if not utiExists then
        tell docRef
          make new uniform type ID with properties {name:"${utiPattern}"}
        end tell
      end if
      
      -- Now find items that conform to this UTI
      set foundItems to (find content under installed items of docRef where UTI conforms to "${utiPattern}")
      repeat with anItem in foundItems
        set itemUTI to UTI of anItem
        set itemInfo to {name:name of anItem, path:posix path of anItem, utiName:name of itemUTI, kind:kind of itemUTI}
        set resultList to resultList & {itemInfo}
      end repeat
      
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

/**
 * Find all sandboxed applications in a package
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - List of sandboxed apps and their entitlements
 */
async function findSandboxedApps(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set appList to (find apps under installed items of first document)
      set resultList to {}
      
      repeat with anApp in appList
        set sandboxEntitlement to entitlement "com.apple.security.app-sandbox" of anApp
        if sandboxEntitlement is not missing value and (sandboxEntitlement requests with allowing) then
          -- This app is sandboxed
          set appInfo to {name:name of anApp, path:posix path of anApp, bundleID:bundle identifier of anApp}
          
          -- Get network entitlements
          set networkClient to entitlement "com.apple.security.network.client" of anApp
          set networkServer to entitlement "com.apple.security.network.server" of anApp
          
          if networkClient is not missing value and (networkClient requests with allowing) then
            set appInfo to appInfo & {clientNetwork:true}
          else
            set appInfo to appInfo & {clientNetwork:false}
          end if
          
          if networkServer is not missing value and (networkServer requests with allowing) then
            set appInfo to appInfo & {serverNetwork:true}
          else
            set appInfo to appInfo & {serverNetwork:false}
          end if
          
          set resultList to resultList & {appInfo}
        end if
      end repeat
      
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  return JSON.parse(result.replace(/[{}]/g, function(match) {
    return match === '{' ? '[' : ']';
  }).replace(/:/g, ','));
}

// Export additional functions
module.exports = {
  findLaunchdJobs,
  findPrivilegedScripts,
  findCriticalIssues,
  getOSRequirements,
  checkOSCompatibility,
  findNonStandardPermissions,
  searchInstallerScripts,
  getComponentPackages,
  findItemsByUTI,
  findSandboxedApps
};