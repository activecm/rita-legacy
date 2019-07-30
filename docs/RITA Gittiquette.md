# RITA Gittiquette:
## Golden Rules:
- Do not split a single piece of work across multiple commits
  - Too many commits makes the git log unreadable
- Do not merge several pieces of work into a single commit
  - Fighting bugs is easier when code changes are logically grouped
- Do not edit the master branch directly
  - The master branch is the sole point of truth for Rita

## Silver Rules:
- Do make sure your public commits leave the code in a working state
  - Fighting bugs is easier when each set of changes is testable
- Do rebase feature branches before submitting a merge request
  - Rebasing solves merge conflicts at the feature branch level
- Do merge branches with a merge commit (using --no-ff)
  - Merge commits create a paper trail and ease the reverting of features

## Bronze Rules:
- Do give your commits meaningful descriptions
  - Knowing what a commit changed without looking at the diff saves time
- Do not give your commits too verbose descriptions
  - Reading a description as long as a code diff takes time
- Do only branch features off of the master branch
  - Branching features off of others complicates the git log and prevents rebasing

## An Overview:
### Creating a feature<sup>[1][3]</sup>:
- Create an issue on the activecm/rita tracker
- Fork activecm/rita to your Github account
- Add your fork of Rita as a remote as shown above.
  - `git remote add development [your git url]`
- Create a feature branch to work on
  - `git branch [your new feature]`
- Checkout the new feature branch
  - `git checkout  [your new feature]`
- Work, commit, test, repeat
  - `git add [files]`
  - `git commit [short descriptive message]`
- Pull down the latest changes in master
  - `git checkout master`
  - `git pull origin master`
- Checkout the feature branch 
  - `git checkout [your new feature]`
- Rebase the feature branch on master<sup>[4]</sup>
  - `git rebase master`
- Push your new commits to Github, remembering to specify your forked copy
  - `git push development [your new feature]`
- Open a merge request using Github’s interface
- Link to the opened issue
  - Example: This merge request addresses issue #34.

### Handling a merge request<sup>[5]</sup>:
- Read through the changes proposed and the linked issue
- Check for correctness
- Check for style
- Check if the feature branch is correctly based on the latest commit of dev
- Checkout the feature branch
  - `git remote add [other user’s name] https://github.com/[other user’s name]/rita.git`
  - `git fetch [other user’s name]`
  - `git checkout [other user’s name]/[feature branch name]`
- Run the code; test out the changes
- Respond with comments, if any, on Github
- Repeat until the changes are acceptable
- Merge with merge commit (no-ff) 
  - Use the merge request process on Github

### Handling a hotfix<sup>[3]</sup>:
- Ensure the hotfix branch fixes the issue
- Merge the hotfix branch into master with a merge commit (no-ff)
  - Use the merge request process on Github or
    - `git checkout master`
    - `git merge [hotfix branch name] --no-ff`
    - `git push origin master`

## Recommended Reading:
1. [A great introduction to git.](http://rogerdudler.github.io/git-guide/)
2. [The git book describes the multi-repository model we have selected for Rita to a tee as an “Integration Manager Workflow.”](https://book.git-scm.com/book/en/v2/Distributed-Git-Distributed-Workflows)
3. [We use the branching model described here without the release branch.](http://nvie.com/posts/a-successful-git-branching-model/)
4. [Rebasing can be conceptually tricky. Atlassian provides a nice write up on the topic.](https://www.atlassian.com/git/tutorials/merging-vs-rebasing)
5. [When merging with Github, you are presented with several options. Github describes these options here.](https://help.github.com/articles/about-pull-request-merges/)
6. [A simple blog post explaining why we use remotes when dealing with git and go code.](http://blog.campoy.cat/2014/03/github-and-go-forking-pull-requests-and.html)
