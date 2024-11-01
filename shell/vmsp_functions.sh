
function enable_vmsp_functions() {
    echo "Enabling VMSP functions"

    function vcompose() {
        vmsp pkg compose -c releases/package.yaml
    }

    function vpush() {
        package_file="${1}"
        vmsp package push localhost:5005 -n vmsp-platform --deploy ${package_file}
    }

    function vbootf() {
        platform_package_file="${1}"
        vmsp bootstrap cluster local --name local-cluster -f ${platform_package_file}
        kswitch vmsp_platform
    }

    function vboot() {
        platform_package_file="${1}"
        vmsp bootstrap cluster local --name local-cluster
        kswitch vmsp_platform
    }
}