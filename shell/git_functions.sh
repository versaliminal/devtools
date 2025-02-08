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

    function git-current-branch() {
        git branch --show-current
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

    function git-changes() {
        git log ${main_branch}..$(git branch --show-current)
    }

    function git-base() {
        git merge-base ${main_branch} $(git branch --show-current)
    }

    function git-fixup() {
        git commit --no-verify -a --fixup=$(git-base)
    }

    function git-squash() {
        commit_msg="$(git log -1 --pretty=%B)"
        git reset --soft $(_git_get_base)
        git commit --amend --allow-empty --no-verify -S -m "${commit_msg}"
    }

    function git-rebase() {
        git fetch && git rebase --no-verify -i origin/${main_branch}
    }

    function git-append() {
        git-fixup && git rebase --no-verify -i --autosquash origin/${main_branch}
    }

    function git-amend() {
        git commit --no-verify --amend --allow-empty
    }

    function git-pub() {
        pre_commit_tests && git push -f
    }

    function git-spub() {
        pre_commit_tests && git-sign && git push -f
    }

    function git-branch() {
        task="${1:-scratch}"
        if [[ `git-branch-exists ${task}` ]]; then
            echo "Branch already exists"
            return 1
        fi
        branch="${user_prefix}/${task}${branch_suffix}"
        git checkout -b ${branch}
        git commit --no-verify --allow-empty -S -m ""
        git push -u origin ${branch}
    }

    function git-reset() {
        git fetch && git reset --hard origin/$(git-current-branch)
    }

    function git-branch-exists() {
        task=$1
        git rev-parse --quiet --verify "${user_prefix}/${task}${branch_suffix}"
    }

    function git-main() {
        git-is-dirty && git stash
        git fetch
        git checkout ${main_branch}
        git reset --hard origin/${main_branch}
    }

    function git-prune() {
        git remote prune origin
        git branch --merged | grep "${user_prefix} | xargs git branch -d"
    }

    function git-is-dirty() {
        ! (git diff-index --quiet --cached HEAD -- && git diff-files --quiet)
    }

    function git-sign() {
        if [[ -z "git config --global user.signkey" ]]; then
            echo "Set a signing key first: git config --global user.signingkey {key}"
            return
        fi
        git commit --amend --no-edit -S --allow-empty
    }

    function pre_commit_tests() {
        # Placeholder
    }

}