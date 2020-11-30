# GTD - Goret Team Deployment -  (🐷)
The main objective of GTD is to simplify the task of GT Releasers team to obtain the status of services and the deployment of processes on AWS ECS services.

## Requirement
GTD requires operators to have access to AWS ECS services, hence, you must have AWS Credentials authorization (AWS ACCESS KEY ID, SECRET ACCESS KEY).  
Since GTD USE DOCKER API to Pull images while deploying on AWS ECR (AWS Docker Registry), Operators needs access to a Docker Server API (Premise Docker, or use of docker context or also set a valid DOCKER_HOST Environment).

## Configuration
Although GTD does not need configuration file to permit status command Requests on AWS ECS Services, you may need some facilities to customize is behavior.

### Config file
The config file, reside on your HOME DIRECTORY, and is named `.gtd.yaml`.

#### Customize the GTD's Behavior

	
		#~/.gtd.yaml
		
		# if you have multiple aws Account and one you prefer to use one by defaut:
		aws_profile: gt
		
		# if you Wanna use a default Deployement environment stack:
		default_env: rct
		
		#if you use Private a Docker Registry
		# Indicate you Docker login
		docker_login: yourdockerlogin
		
		# Do not use this, if you care on Security concerns (prefer the more secure way) 
		docker_password: cleartext password 
		
		# show line number above n element
		table_index_above: 2
		
		# customize the output behavior (allowed values : 'light', 'color')
		table_style: light
	

##### Encrypt your docker hub password with zero knowledge.

Since GTD need credential to pull image From private Docker registry, it permits to store password with a zero knowledge behavior. But, for doing that, it needs some action from you.

You need to export this env on your profile (.bashrc, .zshrc .....)

	export GTD_SHARED_SECRET='Random String'

GTD generates this string for you when it does not found this environment variable and will refuse to create secure env as long as this env is still absent.

When it's done, you can generate your encrypted var with this command :

	gtd add_secret --key docker_password --value your password


## Usage

### Stack description

All GT stacks are described on `gt-release` (Github repository) the one you cloned to read these documents and is stay in each gtd's subdirectory and conforms to the yaml syntax. the file into gtd subdirectory follow this naming convention :

`ENV-PART.yaml`

where ENV-PART could be whatever you wants and represent the stack's name.

The syntax is explicit and human-readable, (unlike the very bad syntax of json file).

Each stack files begin with the name of ECS cluster and its region (AWS meaning) as show below:

```
---
ecs_cluster : ecsClusterName
ecs_region : us-east-1
```

Next come the services description array:

```
services:
  - name: "ECS-SERVICE"
    registry: gutenbergtech/dockerimage
    ignore: true
```

where :
- name: designate the AWS ECS Service Name
- registry: designate the docker image registry
- ignore: flag to toggle the deployement


If one service need to publish docker image on ecr registry while deploying you needs to add `update_ecr` parameter to the service fields and add a repository section:

```
services
  - name: "ECS-SERVICE"
    registry: gutenbergtech/dockerimage
	update_ecr: "whateveryouwant"
	ignore: false
repositories:
  - name: "whateveryouwant"
    repository_name: "ecr/repository/name"
    ignore: false
```

### General execution requirement
while running, GTD look for a subdirectory named `gtd`, where stack files are located.
So execution must occur from the right places.

### Getting Help

`gtd --help`

### Getting the status

If you configured gtd with default stack(env) and or aws profile:

- get the default stack status :

`gtd status`

 output example:
 
 ```
 ┌───┬────────────────────────┬────────────────────────┬──────────┬─────────────────────────────────────────┬────────┬───────────────┐
 │   │ SERVICE                │ FAMILY                 │ REVISION │ CURRENT IMAGE                           │ STATUS │ RUNNING COUNT │
 ├───┼────────────────────────┼────────────────────────┼──────────┼─────────────────────────────────────────┼────────┼───────────────┤
 │ 1 │ svc-recette-hapi       │ tsk-recette-hapi       │       79 │ gutenbergtech/hapi:develop-cbe267d-rct  │ ACTIVE │             1 │
 │ 2 │ svc-recette-hapipusher │ tsk-recette-hapipusher │       80 │ gutenbergtech/hapi:develop-cbe267d-rct  │ ACTIVE │             1 │
 │ 3 │ svc-recette-hapiriver  │ tsk-recette-hapiriver  │       80 │ gutenbergtech/hapi:develop-cbe267d-rct  │ ACTIVE │             1 │
 │ 4 │ svc-recette-hapiws     │ tsk-recette-hapiws     │       80 │ gutenbergtech/hapi:develop-cbe267d-rct  │ ACTIVE │             2 │
 │ 5 │ svc-recette-lms        │ tsk-recette-lms        │       15 │ gutenbergtech/lms:develop-619e3af-rct   │ ACTIVE │             1 │
 │ 6 │ svc-recette-mefio      │ tsk-recette-mefio      │       78 │ gutenbergtech/mefio:develop-8b87381-rct │ ACTIVE │             2 │
 │ 7 │ svc-recette-ap         │ tsk-recette-ap         │       15 │ gutenbergtech/ap:develop-31c68b8-rct    │ ACTIVE │             1 │
 │ 8 │ svc-recette-wr3        │ tsk-recette-wr3        │       58 │ gutenbergtech/wr3:develop-52f7fa2-rct   │ ACTIVE │             1 │
 └───┴────────────────────────┴────────────────────────┴──────────┴─────────────────────────────────────────┴────────┴───────────────┘
 ```

- getting a specific service status:

`gtd status -s svc-recette-lms`

Output example:
```
┌─────────────────┬─────────────────┬──────────┬───────────────────────────────────────┬────────┬───────────────┐
│ SERVICE         │ FAMILY          │ REVISION │ CURRENT IMAGE                         │ STATUS │ RUNNING COUNT │
├─────────────────┼─────────────────┼──────────┼───────────────────────────────────────┼────────┼───────────────┤
│ svc-recette-lms │ tsk-recette-lms │       15 │ gutenbergtech/lms:develop-619e3af-rct │ ACTIVE │             1 │
└─────────────────┴─────────────────┴──────────┴───────────────────────────────────────┴────────┴───────────────┘
```

- getting multiple services:

`gtd status -s svc-recette-lms -s svc-recette-mefio`

Output example:
```
┌───────────────────┬───────────────────┬──────────┬─────────────────────────────────────────┬────────┬───────────────┐
│ SERVICE           │ FAMILY            │ REVISION │ CURRENT IMAGE                           │ STATUS │ RUNNING COUNT │
├───────────────────┼───────────────────┼──────────┼─────────────────────────────────────────┼────────┼───────────────┤
│ svc-recette-lms   │ tsk-recette-lms   │       15 │ gutenbergtech/lms:develop-619e3af-rct   │ ACTIVE │             1 │
│ svc-recette-mefio │ tsk-recette-mefio │       78 │ gutenbergtech/mefio:develop-8b87381-rct │ ACTIVE │             1 │
└───────────────────┴───────────────────┴──────────┴─────────────────────────────────────────┴────────┴───────────────┘
```



### Deploy new Docker Image

#### Deploying a tag

`gtd deploy -t newdockertag`

#### Deploying a new image

- with tag:
`gtd deploy -c gutenbergtech/hapi:dockertag`
- without tag (`:latest`):
`gtd deploy -c gutenbergtech/hapi`
- with both:
`gtd deploy -c gutenbergtech/api -t newdockertag`


to be continued...
