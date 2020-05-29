# RITA Gittiquette

## Commit Guidelines
- Do not split a single piece of work across multiple commits
  - Too many commits makes the git log unreadable
- Do not merge several pieces of work into a single commit
  - Fighting bugs is easier when code changes are logically grouped
- Do give your commits meaningful descriptions
  - Knowing what a commit changed without looking at the diff saves time
- Do not give your commits too verbose descriptions
  - Reading a description as long as a code diff takes time

## Branch Guidelines
- Do not edit the master branch directly
  - The master branch is the sole point of truth for RITA
- Do only branch features off of the master branch
  - Branching features off of others complicates the git log and prevents rebasing
- Do rebase feature branches before submitting a pull request
  - Rebasing solves merge conflicts at the feature branch level

## Github Guidelines
- Do make sure your public commits leave the code in a working state
  - Fighting bugs is easier when each set of changes is testable
- Do merge branches to master with a "squash and merge"
  - Squashing ensures the git history is tidy
  - Merge commits create a paper trail and ease the reverting of features

## Contributors
### Setting up a forked repo
- If you do not have direct write permissions to the RITA project, you will need to [fork it](https://github.com/activecm/rita/fork).
- Once you have a forked repo you will need to clone it to a very specific path which corresponds to _the original repo location_. This is due to the way packages are imported in Go programs.
  - `git clone [your forked repo git url]`
- Add `https://github.com/activecm/rita` as a new remote so you can pull new changes.
  - `git remote add upstream https://github.com/activecm/rita`

### Creating a feature<sup>[1]</sup>
- Create an issue on the activecm/rita tracker
- Create a feature branch to work on
  - `git branch [your new feature]`
- Checkout the new feature branch
  - `git checkout  [your new feature]`
- Work, commit, test, repeat
  - `git add [files]`
  - `git commit [short descriptive message]`
- Pull down the latest changes in upstream master
  - `git checkout master`
  - `git pull -r upstream master`
- Rebase the feature branch on master<sup>[2]</sup>
  - `git checkout [your new feature]`
  - `git rebase master`
- Push your new commits to Github
  - `git push origin [your new feature]`
- Open a pull request using Github’s interface

## Maintainers
### Handling a pull request<sup>[3]</sup>
- Read through the changes proposed and the linked issue
- Check for correctness
- Check for style
- Check if the feature branch has only the intended commits and doesn't cause any merge conflicts
- Checkout the feature branch
  - `git remote add [other username] https://github.com/[other username]/rita.git`
  - `git fetch [other username]`
  - `git checkout [other username]/[feature branch name]`
  - `git checkout -b [feature branch name]`
    - Note: this step is optional but lets you directly make changes to the contributor's repo using `git push [other username] [feature branch name]`
- Run the code; test out the changes
- Leave a code review with comments, requested changes, or an approval on Github
- Optionally, make changes yourself and push to the contributor's branch
- Merge with the "Squash and merge" option, leaving a descriptive comment in the commit message
- Copy the descriptive comment to the latest [release draft](https://github.com/activecm/rita/releases)

## Recommended Reading
1. [A great introduction to git.](http://rogerdudler.github.io/git-guide/)
2. [Rebasing can be conceptually tricky. Atlassian provides a nice write up on the topic.](https://www.atlassian.com/git/tutorials/merging-vs-rebasing)
3. [When merging with Github, you are presented with several options. Github describes these options here.](https://help.github.com/en/github/collaborating-with-issues-and-pull-requests/about-pull-request-merges)
