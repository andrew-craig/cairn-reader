# Cairn Project Tasks

This folder contains tasks to improve the Cairn project. It is a simple, light-weight alternative to Jira tasks or Github issues.

## Guidelines

- Each task file contains a complete problem description and implementation guide
- Each file name starts iwth the priority of the task: 
    - P0: critical tasks, must be implemented
    - P1: high priority tasks to be implemented as soon as possible
    - P2: medium priority tasks, required for production launch, but not beta
    - P3: low priority, nice to haves.

Use `tree -L 1 todos/ -h --charset ascii` to see the list of tasks

## How to Use

1. **Pick a task:** Choose from the lists above
2. **Review the task file:** Read the problem, impact, and proposed solution
3. **Implement the solution:** Follow the steps and guidance in the task file
4. **Test thoroughly:** Use the testing strategy outlined in the task. Verify changes match the task requirements
5. **Move to done:** When done, move the file to the @/todos/done folder
