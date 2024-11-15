## AutoCheckBot

<p> The function of this bot that it collects data from user which stores in JSON file. </p>

<p>Data stored in this format: </p>

```json

{
    "groups": {
        "Group№1": {
            "relevance": true,
            "users": [
                {
                    "login": "example@gmail.com",
                    "password": "hash-of-password",
                    "subscription": false
                }
            ]
        }
    }
}

```

<h3>Export env variables</h3>

```
export token=""
export channel=""

```


<h3>To run bot</h3>    

default way: ``` make run ``` 

To make image/docker: ```make image --> make docker```


