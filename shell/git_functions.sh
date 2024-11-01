main_branch="main"
user_prefix="vmsp/dmccaffrey"

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

    function get_base() {
        git merge-base ${main_branch} $(git branch --show-current)
    }

    function fixup() {
        git commit --no-verify -a --fixup=$(get_base)
    }

    function asquash() {
        git rebase --no-verify -i --autosquash origin/${main_branch}
    }

    function squash() {
        commit_msg="$(git log -1 --pretty=%B)"
        git reset --soft $(get_base)
        git commit --amend --allow-empty --no-verify -S -m "${commit_msg}"
    }

    function rebase() {
        git fetch && git rebase --no-verify -i origin/${main_branch}
    }

    function amend() {
        git commit --no-verify --amend --allow-empty
    }

    function pub() {
        pre_commit_tests && git push -f
    }

    function append() {
        fixup && squash
    }

    function branch() {
        git_reset_master
        git_branch "$1" "$2"
    }

    function spub() {
        pre_commit_tests && sign_commit && git push -f
    }

    function git_branch() {
        task="${1:-scratch}"
        if [[ `git_branch_exists ${task}` ]]; then
            echo "Branch already exists"
            return 1
        fi
        branch="${user_prefix}/${task}"
        git checkout -b ${branch}
        commit_msg=$(printf "feat: TODO summary\n\nTODO description\n\n**Change Submission Checklist**\n- [ ] Feature Added/Updated\n- [ ] Packaging Added/Updated\n- [ ] Unit Tests Added/Updated\n- [ ] Docs Added/Updated\n\nTesting performed:\nTODO\n\n${task}")
        git commit --no-verify --allow-empty -S -m ""
        git commit --amend --allow-empty --no-verify -S -m "${commit_msg}"
        git push -u origin ${branch}
    }

    function git_reset() {
        resetmaster && git clean -d force
    }

    function git_branch_exists() {
        task=$1
        git rev-parse --quiet --verify "${user_prefix}/${task}"
    }

    function git_reset_master() {
        git_is_dirty && git stash
        git fetch
        git checkout ${main_branch}
        git reset --hard origin/${main_branch}
    }

    function git_prune() {
        git remote prune origin
        git branch --merged | grep dmcc/ | xargs git branch -d
    }

    function git_task() {
        branch=$(git branch --show-current)
        task=${branch#dmcc/}
        echo ${task}
    }

    function git_is_dirty() {
        ! (git diff-index --quiet --cached HEAD -- && git diff-files --quiet)
    }

    function enable_gpg() {
        git config --global user.signingkey ${1}
    }

    function sign_commit() {
        git commit --amend --no-edit -S --allow-empty
    }

    function pre_commit_tests() {
        #pre-commit run --all-files
        pre-commit run -v
    }

}