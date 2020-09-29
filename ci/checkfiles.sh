# Base Branch
if [ -f ".git/resource/metadata.json" ]; then
    BASE_BRANCH="$(jq -r '.[] | select (.name == "base_sha").value' .git/resource/metadata.json)"
else
    BASE_BRANCH="$(git config --get pullrequest.basebranch)"
fi

echo $BASE_BRANCH

git --no-pager diff --name-only --diff-filter=ACMR "${BASE_BRANCH}" > changed_files.txt

# Checking shell scripts
printf "**** Checking shell files ****\\n"
mkfifo check_files
grep -i -E '\.sh$' changed_files.txt > check_files &
while IFS= read -r FILE
do
    if ! shellcheck "${FILE}"; then
        INVALID="${INVALID}${FILE}\\n"
    fi
done < check_files
rm check_files

if [ -n "${INVALID}" ]; then
    echo "The following files contain invalid syntax:"
    # shellcheck disable=SC2059
    printf "${INVALID}"
    exit 1
fi

echo "No files with invalid syntax detected"