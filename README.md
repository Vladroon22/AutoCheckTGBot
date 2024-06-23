## AutoCheckBot

<p>The function of this bot that it collects data from user which stores in JSON file. </p>

<p>Data stored in this format: </p>
```json
{
    "groups": {
        "Group№1": {
            "relevance": true,
            "users": [
                {
                    "login": "example@gmail.com",
                    "password": "....",
                    "subscription": false
                }
            ]
        }
    }
}
```

<p> To run bot: </p> ```make run```

<p> To make image/docker: </p> ```make image --> make docker```
