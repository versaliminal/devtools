---
name: create-task
description: Create and track a new task
license: MIT
compatibility: opencode
metadata:
  audience: developers
  workflow: general
---

## What I do

- Create and track new tasks locally in markdown files in the vimwiki directory.
- Create a directory for the current project under the directory ~/Documents/vimwiki/opencode/ if one does not already exist.
- Create a new markdown file named 'tasks.md'to contain tasks for the project if one does not exist.
- If the file was created add a link to the new file in the index.md under the heading 'opencode tasks' with the format:
  - [Project Name](opencode/ProjectName/tasks.md)
- Search the tasks.md file for the current task name. If it does not exist, add a new heading entry for the task with the current date and time.
- Created an entry for each sub-task for the primary task in the format:
  - [ ] Sub-task name: sub-task description
- If the task already exists, add a new entry for the sub-task under the existing task heading.
- If a sub-task already exists, do not add it again.
- If a task is marked as completed, do not add new sub-tasks to it, create a new task instead.
- If a sub-task is marked as completed it may be marked incomplete again if needed, but it should not be duplicated.
- A completed task or sub-task is indicated by changing the checkbox from [ ] to [x]. They should never be deleted, only marked as completed or incomplete.

## When to use me

Use this when you are starting work on a new task or project and and the work is non-trivial.
Ask clarifying questions if the inclusion of a task or sub-task is not clear.
Ask clarifying questions if the state of a task is not clear.