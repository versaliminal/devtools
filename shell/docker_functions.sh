function enable_docker_functions() {

    echo "Enabling docker functions"

    function docker-clean() {
        docker rm `docker ps -q -f status=exited`
        docker rmi $(docker images -f "dangling=true" -q)
    }
}