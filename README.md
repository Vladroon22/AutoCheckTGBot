## AutoCheckBot

<h3> The function of this bot that it collects data from user which stores in MongoDB. </h3>

```
[
  {
    _id: ObjectId('....'),
    groupname: 'qwe',
    login: 'qwe',
    hash: '$2a$10$Z1VYfo0QwbqZxAoaNdZ0zOjr0GOQqkvuSYe7qN232Zkr6j6Ee8aky',
    subscription: true
  }
]

```

<h3>Export env variables in .env-file</h3>

```
token=""
channel=""
```

<h3>To run bot</h3>    

```
sudo docker run --name=my-mongo -p 27017:27017 -d mongo:8.0
```

```
make run
``` 

<h4>To make image/docker </h4>

```
make image
```

```
make docker
```

