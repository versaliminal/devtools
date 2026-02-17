main_branch="main"
user_prefix="qm"
branch_suffix=""

# Example workflow:
#   branch task-xyz
#   <make changes>
#   append
#   <make changes>
#   append
#   rebase
#   amend <describe changes>
#   pub

export GPG_TTY=$(tty)

function enable_git_functions() {
    echo "Enabling git functions"

    RED='\033[0;31m'
    NC='\033[0m' # No Color

    function _indent() {
        sed 's/^/ - /';
    }

    function git-status() {
        echo "${RED}Status:${NC}"
        git -c color.ui=always status | _indent

        echo "${RED}Commits:${NC}"
        current_branch="$(git-current-branch)"
        git -c color.ui=always log --oneline ${main_branch}..${current_branch} | _indent

        echo "${RED}Commited files:${NC}"
        git diff --name-only ${current_branch} $(git-base) | _indent
    }

    function git-prune() {
        git remote prune origin
        git branch --merged | grep "${user_prefix} | xargs git branch -d"
    }

    function git-is-dirty() {
        ! (git diff-index --quiet --cached HEAD -- && git diff-files --quiet)
    }

    function pre_commit_tests() {
        # Placeholder
    }

}