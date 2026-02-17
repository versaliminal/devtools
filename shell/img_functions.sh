function enable_img_functions() {
    echo "Enabling image functions"

    function minimg() {
        source_img=${1}
        filename=$(basename -- "${source_img}")
        filename="${filename%.*}"
        magick "${source_img}" -resize 800x600\> -depth 8 jpg:${filename}-min.jpg
    }
}