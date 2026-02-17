# Depends on the git plugin for work_in_progress()
(( $+functions[work_in_progress] )) || work_in_progress() {}

ZSH_THEME_GIT_PROMPT_PREFIX="%{$reset_color%}%{$fg[white]%}⌊ %{$fg[yellow]%}"
ZSH_THEME_GIT_PROMPT_SUFFIX="%{$fg[white]%} ⌋%{$reset_color%}"
ZSH_THEME_GIT_PROMPT_DIRTY="⌊%{$fg[red]%} ⌧ %{$reset_color%}⌋"
ZSH_THEME_GIT_PROMPT_CLEAN=" "

# Customized git status, oh-my-zsh currently does not allow render dirty status before branch
git_custom_status() {
  local branch=$(git_current_branch)
  [[ -n "$branch" ]] || return 0
  echo "$(parse_git_dirty)\
%{${fg_bold[yellow]}%}$(work_in_progress)%{$reset_color%}\
${ZSH_THEME_GIT_PROMPT_PREFIX}${branch}${ZSH_THEME_GIT_PROMPT_SUFFIX}"
}

# Combine it all into a final right-side prompt
RPS1="⌋\$(git_custom_status)"
PROMPT='%{$fg[white]%}⌊ %{$fg[yellow]%}%~% %{$fg[white]%} ⌋⌊%(?.☈↠.☈) %{$reset_color%}'
