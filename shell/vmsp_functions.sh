
function enable_vmsp_functions() {
    echo "Enabling VMSP functions"

    function vcompose() {
        vmsp pkg compose -c releases/package.yaml
    }

    function vpush() {
        local package_file="${1}"
        vmsp pkg push --allow-insecure-registry localhost:50000 "${package_file}"
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

    function bundle-get() {
        k8s_token=$(kubectl get secrets synthetic-checker-krp -n vmsp-platform -ojsonpath={.data.token} | base64 -d)
        ingress_address="$(kubectl get gateway vmsp-gateway -n istio-ingress -ojson | jq -r '.status.addresses[0].value')"
        echo "Triggering bundle generation"
        local job_id=$(curl -s -k -XPOST -H "Authorization: Bearer $k8s_token" "https://${ingress_address}:30005/webhooks/vmsp-platform/supportbundle/generate" -d '{}' | jq -r '.id')
        echo "Waiting on job ${job_id} to complete"
        local job_status=$(curl -s -k -XGET -H "Authorization: Bearer ${k8s_token}" "https://${ingress_address}:30005/webhooks/vmsp-platform/supportbundle/generate/${job_id}" | jq '.completedAt')
        while [[ ${job_status} == 'null' ]]; do
            sleep 15
            job_status=$(curl -s -k -XGET -H "Authorization: Bearer ${k8s_token}" "https://${ingress_address}:30005/webhooks/vmsp-platform/supportbundle/generate/${job_id}" | jq '.completedAt')
            echo "CompletedAt: ${job_status}"
        done
        local bundle_link=$(curl -s -k -XGET -H "Authorization: Bearer ${k8s_token}" "https://${ingress_address}:30005/webhooks/vmsp-platform/supportbundle/generate/${job_id}" | jq -r '.outputValues.download_url')
        local bundle_file="${bundle_link##*/}"
        echo "Downloading bundle ${bundle_file}"
        curl -X GET -k -v -u "vmware-system-user:${VSPHERE_PASSWORD}" "http://${ingress_address}/supportbundles/${bundle_file}"
        echo "Job is complete"
    }

    function bundle-list() {
        k8s_token=$(kubectl get secrets synthetic-checker-krp -n vmsp-platform -ojsonpath={.data.token} | base64 -d)
        ingress_address="$(kubectl get gateway vmsp-gateway -n istio-ingress -ojson | jq -r '.status.addresses[0].value')"
        curl -k -XPOST -H "Authorization: Bearer $k8s_token" https://${ingress_address}:30005/webhooks/vmsp-platform/supportbundle/list
    }

    function node-ssh() {
        echo "Pass = ${APPLIANCE_ROOT_PASSWORD}"
        for node in $(kubectl get nodes -o name); do
            node_ip=$(kubectl get -o json "${node}" | jq -r '.status.addresses[] | select (.type == "ExternalIP") | .address')
            echo "${node} -- ssh vmware-system-user@${node_ip}"
        done
    }

    function worker-extend() {
        USER=dm020578 /mts/git/bin/nimbus-ctl --lease 7 extend-lease dm020578-worker
    }

    function nimbus-deploy() {
        time python3 utils/deploy_testbed.py --deploy-input ~/deploy_input.json --deploy-vcf --deploy-vmsp > ~/vcf_vmsp.log 2>&1 &
    }
}