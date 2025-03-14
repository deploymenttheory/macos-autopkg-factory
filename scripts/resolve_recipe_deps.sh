#!/bin/bash

set -euo pipefail

AUTOPKG_PREFS_PATH="${AUTOPKG_PREFS_PATH:-$HOME/Library/Preferences/com.github.autopkg.plist}"
DEFAULT_RECIPE_LIST="${GITHUB_WORKSPACE}/configuration/recipe_list.txt"
REPO_LIST_PATH="./configuration/repo_list.txt"

ARGS=(--prefs="$AUTOPKG_PREFS_PATH" --use-token --repo-list-path="$REPO_LIST_PATH")

echo "🔍 Resolving recipe repository dependencies..."

if [[ -n "${RECIPE:-}" ]]; then
    echo "📌 Using single recipe: ${RECIPE}"
    ARGS+=(--recipes="${RECIPE}")

elif [[ -n "${RECIPES:-}" ]]; then
    echo "📌 Multiple recipes provided: ${RECIPES}"
    ARGS+=(--recipes="${RECIPES}")

elif [[ -n "${RECIPE_LIST_FILE:-}" ]]; then
    echo "📌 Recipe list file provided: ${RECIPE_LIST_FILE}"
    if [[ -f "${RECIPE_LIST_FILE}" ]]; then
        RECIPES_PARSED=$(grep -v '^\s*#' "${RECIPE_LIST_FILE}" | xargs | sed 's/ /,/g')
        echo "📋 Parsed recipes from file: ${RECIPES_PARSED}"
        ARGS+=(--recipes="${RECIPES_PARSED}")
    else
        echo "❌ Recipe list file not found: ${RECIPE_LIST_FILE}"
        exit 1
    fi

else
    echo "📌 No inputs provided. Using default recipe list: ${DEFAULT_RECIPE_LIST}"
    if [[ -f "${DEFAULT_RECIPE_LIST}" ]]; then
        DEFAULT_RECIPES=$(grep -v '^\s*#' "${DEFAULT_RECIPE_LIST}" | xargs | sed 's/ /,/g')
        echo "📋 Parsed default recipes: ${DEFAULT_RECIPES}"
        ARGS+=(--recipes="${DEFAULT_RECIPES}")
    else
        echo "❌ Default recipe list file not found: ${DEFAULT_RECIPE_LIST}"
        exit 1
    fi
fi

# Run the autopkgctl command with dynamically built args
autopkgctl recipe-repo-deps "${ARGS[@]}"
