## AutoCheckBot

<p>The main function of this TG-Bot. It collects data from user which save in json file. </p>

<p>Data stored in this format: </p>
```
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
