# asana-go-exporter
Asana projects tasks exporter


## Install

 1. Download project
 2. Enter required .env file information:
    - workspace
    - asana access token : https://app.asana.com/0/my-apps
    - enter backup destination

## Run

```bash
$ go run *.go
```

## Result

Application will create file/folder structure simillar to that in the asana workspace enviroment.

All projects, tasks, stories, and assets data is stored in parsable json files.

Assetes are downloaded in `assets` directory in each task folder.


 - my-workspace
   - my-workspace.json
   - projects
     - my-project-name1
       - my-project-name1.json
       - my-project-name1-task-list.json
       - tasks
         - 123123123123-taskid
           - 123123123123-task.json
           - 123123123123-story.json
           - assets
             - assets_list.json
             - 789789789789.png
             - 321321321321.jpg
         - 456456456546-taskid
           - 456456456546-task.json
           - 456456456546-story.json
     - my-project-name1
       - ...
## Resumable

Since non-paid version of asana wont let you do more than 50 requests per minute, downloading all tasks can take a long time.

All projects, tasks, stories are stored in json files, that way application can check missing files and continue if the process was interrupted.

## Changes tracking

- Useful for tracking incognito changes on task story records.
  - At my former company, there was that one guy that would "perfecting" his story comments. This comment edit/change does not mark the entire task as changed, only a small "edit" mark in the story/comment header. So some stupid remarks would over time convert to magic and futuristic predictions :-). (I ran backup/compare every week for several years, it is interesting to see how some people struggle to be always right.)


## Todo

 - Build MD files for each task/story
