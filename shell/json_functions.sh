function enable_json_functions() {
    echo "Enabling JSON functions"

    function jpp() {
        file="${1}"
        jq "." ${file}
    }

    function jselect() {
        query="${1}"
        for file in *.json; do
            if [[ `jq ${file} ${query}` == 'true' ]]; then
                echo "${file}"
            fi
        done
    }

    function jquery() {
        query="${1}"
        for file in *.json; do
            echo -e "\nFile: ${file}"
            jq ${file} ${query}
        done
    }
}