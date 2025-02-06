
function enable_k8s_functions() {
    echo "Enabling k8s functions"

    alias kub="kubectl"

    function kub-ns() {
        local namespace="${1}"
        kubectl config set-context --current --namespace=${namespace}
    }

    function kub-stat() {
        kubectl get hr
    }

    function kub-conf() {
        kubectl config view --minify
    }

    function kub-pods() {
        local search_string="${1}"
        if [[ -n "${search_string}" ]]; then
            kubectl get pods -o wide | grep "${search_string}"
        else
            kubectl get pods -o wide
        fi
    }

    function kub-jobs() {
        kubectl get jobs
        kubectl get cronjobs
    }

    function kub-j2p() {
        local job_name="${1}"
        kubectl get pods -l job-name=${job_name} -o name
    }

    function kub-jlogs() {
        local job_name="${1}"
        kubectl logs -f $(kub-j2p "${job_name}")
    }
}