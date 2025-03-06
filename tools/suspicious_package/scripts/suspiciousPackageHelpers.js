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
    // For debugging
    console.error("Running AppleScript:\n", script.substring(0, 300) + (script.length > 300 ? "..." : ""));
    
    exec(`osascript -e '${script.replace(/'/g, "'\\''")}'`, (error, stdout, stderr) => {
      if (error) {
        console.error("AppleScript error:", error.message);
        console.error("stderr:", stderr);
        reject(new Error(`AppleScript execution error: ${error.message}`));
        return;
      }
      if (stderr) {
        console.error("AppleScript stderr:", stderr);
      }
      console.error("AppleScript stdout:", stdout.substr(0, 100) + (stdout.length > 100 ? "..." : ""));
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
  if (result && result.trim()) {
    return JSON.parse(result.replace(/[{}]/g, function(match) {
      return match === '{' ? '[' : ']';
    }).replace(/:/g, ','));
  } else {
    return []; // Return empty array instead of parsing empty string
  }
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
  if (result && result.trim()) {
    return JSON.parse(result.replace(/[{}]/g, function(match) {
      return match === '{' ? '[' : ']';
    }).replace(/:/g, ','));
  } else {
    return []; // Return empty array instead of parsing empty string
  }
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
      
      -- Return values individually to avoid complex string concatenation
      return sigStatus & "," & isNotarized
    end tell
  `;
  
  try {
    const result = await runAppleScript(script);
    if (result && result.trim()) {
      const parts = result.split(',');
      return {
        name: packagePath.split('/').pop(),
        signatureStatus: parts[0],
        notarized: parts[1] === "true"
      };
    }
  } catch (e) {
    console.error("Failed to get signature info:", e);
  }
  
  return { 
    name: packagePath.split('/').pop(), 
    signatureStatus: "unknown", 
    notarized: false 
  };
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
  if (result && result.trim()) {
    return JSON.parse(result.replace(/[{}]/g, function(match) {
      return match === '{' ? '[' : ']';
    }).replace(/:/g, ','));
  } else {
    return []; // Return empty array instead of parsing empty string
  }
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
  if (result && result.trim()) {
    return JSON.parse(result.replace(/[{}]/g, function(match) {
      return match === '{' ? '[' : ']';
    }).replace(/:/g, ','));
  } else {
    return []; // Return empty array instead of parsing empty string
  }
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
  if (result && result.trim()) {
    return JSON.parse(result.replace(/[{}]/g, function(match) {
      return match === '{' ? '[' : ']';
    }).replace(/:/g, ','));
  } else {
    return []; // Return empty array instead of parsing empty string
  }
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
        -- Get supported architectures directly
        set archList to supported architectures of codeItem
        set doesSupport to false
        
        -- Check if the architecture is in the supported architectures list
        if archList contains "${architecture}" then
          set doesSupport to true
        end if
        
        set itemInfo to {name:name of codeItem, path:posix path of codeItem, supports:doesSupport}
        set resultList to resultList & {itemInfo}
      end repeat
      return resultList
    end tell
  `;
  
  try {
    const result = await runAppleScript(script);
    if (result && result.trim()) {
      return JSON.parse(result.replace(/[{}]/g, function(match) {
        return match === '{' ? '[' : ']';
      }).replace(/:/g, ','));
    }
  } catch (e) {
    console.error("Failed to check architecture support:", e);
  }
  
  return []; // Return empty array instead of parsing empty string
}

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

      -- Handle empty case
      if (count of launchdJobs) = 0 then
        return "[]"
      end if

      -- Convert AppleScript list to JSON format
      set resultList to ""
      repeat with aJob in launchdJobs
        -- Parse ownerID to ensure it's a number
        set ownerId to owner ID of aJob
        if ownerId is missing value or ownerId is "" then
          set ownerId to 0
        end if
        
        set jobInfo to "{\\"name\\": \\"" & name of aJob & "\\", \\"path\\": \\"" & posix path of aJob & "\\", \\"owner\\": \\"" & owner of aJob & "\\", \\"ownerID\\": " & ownerId & ", \\"permissions\\": \\"" & symbolic permissions of aJob & "\\"}"
        set resultList to resultList & jobInfo & ","
      end repeat

      return "[" & text 1 thru -2 of resultList & "]"
    end tell
  `;

  const result = await runAppleScript(script);

  if (result && result.trim()) {
    try {
      return JSON.parse(result);
    } catch (error) {
      console.error("❌ Failed to parse AppleScript output:", result);
      throw new Error("Invalid JSON returned from AppleScript");
    }
  } else {
    return []; // Return empty array instead of parsing empty string
  }
}

/**
 * Finds the number of installer scripts that run as root and their names
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - Array of script metadata (name, short name, runs when)
 */
async function findInstallerScriptsRunAsRoot(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set rootScripts to installer scripts of first document whose runs as user is "root"

      -- Handle empty list case
      if (count of rootScripts) = 0 then
        return "[]"
      end if

      -- Convert script names to JSON format
      set resultList to "["
      repeat with aScript in rootScripts
        set scriptName to name of aScript
        set shortName to short name of aScript
        set whenRun to runs when of aScript

        set resultList to resultList & "{\\"name\\": \\"" & scriptName & "\\", \\"shortName\\": \\"" & shortName & "\\", \\"when\\": \\"" & whenRun & "\\"},"
      end repeat

      -- Remove trailing comma and close JSON array
      set resultList to text 1 thru -2 of resultList & "]"
      return resultList
    end tell
  `;

  const result = await runAppleScript(script);

  if (result && result.trim()) {
    try {
      return JSON.parse(result);
    } catch (error) {
      console.error("❌ Failed to parse AppleScript output:", result);
      throw new Error("Invalid JSON returned from AppleScript");
    }
  } else {
    return []; // Return empty array instead of parsing empty string
  }
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
      
      -- Handle empty list case
      if (count of criticalIssues) = 0 then
        return "[]"
      end if
      
      -- Convert to JSON format directly
      set resultList to "["
      repeat with anIssue in criticalIssues
        set issueDetails to details of anIssue
        set issuePriority to priority of anIssue
        
        set jsonObj to "{\\"details\\": \\"" & issueDetails & "\\", \\"priority\\": \\"" & issuePriority & "\\""
        
        -- Add path if it exists
        set relPath to related POSIX path of anIssue
        if relPath is not missing value then
          set jsonObj to jsonObj & ", \\"path\\": \\"" & relPath & "\\""
        end if
        
        set resultList to resultList & jsonObj & "},"
      end repeat
      
      -- Remove trailing comma and close JSON array
      set resultList to text 1 thru -2 of resultList & "]"
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  
  if (result && result.trim()) {
    try {
      return JSON.parse(result);
    } catch (error) {
      console.error("❌ Failed to parse AppleScript output:", result);
      throw new Error("Invalid JSON returned from AppleScript");
    }
  } else {
    return []; // Return empty array instead of parsing empty string
  }
}

/**
 * Gets the minimum OS version requirements for all executable components in a package
 * @param {string} packagePath - Full path to the package file
 * @returns {Promise<Array>} - Array of executables with their OS version requirements
 */
async function getMacOSMinimumVersionRequirements(packagePath) {
  const script = `
    tell application "Suspicious Package"
      open POSIX file "${packagePath}"
      set codeItems to (find code under installed items of first document)
      
      -- Prepare result as JSON array
      set resultList to "["
      
      repeat with anItem in codeItems
        set minOS to OS minimum version of anItem
        if minOS is not missing value then
          set itemName to name of anItem
          set itemPath to POSIX path of anItem
          set osPlatform to platform of minOS
          set osVersion to product version of minOS
          set osMajor to major version number of minOS

          -- Construct JSON object
          set jsonEntry to "{\\"name\\": \\"" & itemName & "\\", \\"path\\": \\"" & itemPath & "\\", \\"platform\\": \\"" & osPlatform & "\\", \\"version\\": \\"" & osVersion & "\\", \\"major\\": " & osMajor & "}"

          -- Append to result list
          set resultList to resultList & jsonEntry & ","
        end if
      end repeat

      -- Close JSON array, handling empty case
      if resultList is "[" then
        set resultList to "[]"
      else
        set resultList to text 1 thru -2 of resultList & "]"
      end if

      return resultList
    end tell
  `;

  const result = await runAppleScript(script);
  
  if (result && result.trim()) {
    try {
      return JSON.parse(result);
    } catch (error) {
      console.error("❌ Failed to parse AppleScript output:", result);
      throw new Error("Invalid JSON returned from AppleScript");
    }
  } else {
    return []; // Return empty array instead of parsing empty string
  }
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
      set permItems to (find content under installed items of first document where everyones privileges is not read only or owner is not "root")
      
      -- Convert AppleScript list to JSON format
      set resultList to ""
      repeat with anItem in permItems
        set itemInfo to "{\\"name\\": \\"" & name of anItem & "\\", \\"path\\": \\"" & posix path of anItem & "\\", \\"symPerm\\": \\"" & symbolic permissions of anItem & "\\", \\"owner\\": \\"" & owner of anItem & "\\", \\"group\\": \\"" & group of anItem & "\\"}"
        set resultList to resultList & itemInfo & ","
      end repeat
      
      if length of resultList > 0 then
        return "[" & text 1 thru -2 of resultList & "]"
      else
        return "[]"
      end if
    end tell
  `;

  const result = await runAppleScript(script);

  if (result && result.trim()) {
    try {
      return JSON.parse(result);
    } catch (error) {
      console.error("❌ Failed to parse AppleScript output:", result);
      throw new Error("Invalid JSON returned from AppleScript");
    }
  } else {
    return []; // Return empty array instead of parsing empty string
  }
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
  if (result && result.trim()) {
    return JSON.parse(result.replace(/[{}]/g, function(match) {
      return match === '{' ? '[' : ']';
    }).replace(/:/g, ','));
  } else {
    return []; // Return empty array instead of parsing empty string
  }
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
  
  // For debugging
  console.error("Running getComponentPackages script");
  
  try {
    const result = await runAppleScript(script);
    console.error("Raw AppleScript result:", result);
    
    if (result && result.trim()) {
      return JSON.parse(result.replace(/[{}]/g, function(match) {
        return match === '{' ? '[' : ']';
      }).replace(/:/g, ','));
    } else {
      return []; // Return empty array instead of parsing empty string
    }
  } catch (error) {
    console.error("Error in getComponentPackages:", error);
    return [];
  }
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
  if (result && result.trim()) {
    return JSON.parse(result.replace(/[{}]/g, function(match) {
      return match === '{' ? '[' : ']';
    }).replace(/:/g, ','));
  } else {
    return []; // Return empty array instead of parsing empty string
  }
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
      
      -- Handle empty case
      if (count of appList) = 0 then
        return "[]"
      end if
      
      -- Convert to JSON format directly
      set resultList to "["
      
      repeat with anApp in appList
        set sandboxEntitlement to entitlement "com.apple.security.app-sandbox" of anApp
        if sandboxEntitlement is not missing value and (sandboxEntitlement requests with allowing) then
          -- This app is sandboxed
          set appName to name of anApp
          set appPath to posix path of anApp
          set appBundleID to bundle identifier of anApp
          
          -- Check network entitlements
          set hasClientNetwork to false
          set hasServerNetwork to false
          
          set networkClient to entitlement "com.apple.security.network.client" of anApp
          if networkClient is not missing value and (networkClient requests with allowing) then
            set hasClientNetwork to true
          end if
          
          set networkServer to entitlement "com.apple.security.network.server" of anApp
          if networkServer is not missing value and (networkServer requests with allowing) then
            set hasServerNetwork to true
          end if
          
          -- Create JSON object for this app
          set jsonObj to "{\\"name\\": \\"" & appName & "\\", \\"path\\": \\"" & appPath & "\\", \\"bundleID\\": \\"" & appBundleID & "\\", \\"clientNetwork\\": " & hasClientNetwork & ", \\"serverNetwork\\": " & hasServerNetwork & "}"
          
          set resultList to resultList & jsonObj & ","
        end if
      end repeat
      
      -- Handle empty results or remove trailing comma
      if resultList is "[" then
        return "[]"
      else
        set resultList to text 1 thru -2 of resultList & "]"
      end if
      
      return resultList
    end tell
  `;
  
  const result = await runAppleScript(script);
  
  if (result && result.trim()) {
    try {
      return JSON.parse(result);
    } catch (error) {
      console.error("❌ Failed to parse AppleScript output:", result);
      throw new Error("Invalid JSON returned from AppleScript");
    }
  } else {
    return []; // Return empty array instead of parsing empty string
  }
}

// Export all functions in a single module.exports
module.exports = {
  // Original functions
  openPackage,
  getInstalledItems,
  getInstalledApps,
  checkPackageSignature,
  getInstallerScripts,
  findPackageIssues,
  findComponentsWithEntitlement,
  exportDiffableManifest,
  checkArchitectureSupport,
  findLaunchdJobs,
  findInstallerScriptsRunAsRoot,
  findCriticalIssues,
  getMacOSMinimumVersionRequirements,
  findNonStandardPermissions,
  searchInstallerScripts,
  getComponentPackages,
  findItemsByUTI,
  findSandboxedApps
};