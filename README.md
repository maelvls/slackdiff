# Slack Diff, know what channels you are missing on!

## Install

```sh
(cd && GO111MODULE=on go get github.com/maelvls/slackdiff@master)
```

## Help

```sh
% slackdiff --help
Usage of slackdiff:
  -token string
        Slack OAuth token, create one at https://api.slack.com/apps. You can also pass it with SLACK_TOKEN. The --token has priority over SLACK_TOKEN.
  -users string
        Comma-separated list of users to select. By default, shows all users.
  -version
        Watch out, returns 'n/a (commit none, built on unknown)' when built with 'go get'.

Environment variables:
  SLACK_TOKEN
        Slack OAuth token, see https://api.slack.com/apps.exit status 2
```
