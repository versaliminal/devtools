#!/bin/bash

brew bundle --file=~/Brewfile

cp terminal/zsh/zshrc ~/.zshrc
cp vim/vimrc ~/.vimrc
cp terminal/ghostty/config $HOME/Library/Application\ Support/com.mitchellh.ghostty/config
cp git/gitconfig ~/.gitconfig
ln -s $(pwd)/markdown/templates ~/.md