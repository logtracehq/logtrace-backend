- create a finger print of the browser when the user visits a website 
- store it in the database 
- when they come back again, check if the device finger print exists 
- if not, create a record in the database 
- if yes, retrive the records and reuse them in the succeeding requests


{
    "id": "b8fc8dbb-e946-4dce-b7d8-6420d2e80b0a",
    "ipAddress": "41.84.201.115",
    "userAgent": "Mozilla/5.0 (X11; Linux x86_64; rv:145.0) Gecko/20100101 Firefox/145.0",
    "userAgentType": "web",
    "expiresAt": "2025-12-10T23:14:16.252Z",
    "createdAt": "2025-12-10T23:14:16.253Z",
    "updatedAt": "2025-12-10T23:14:16.253Z",
    "orgId": "8f7da686-0f49-4e9c-8cee-d7cbd19217be",
    "projectId": "be77b3c5-1f49-43de-8004-ae5f27403e26",
    "projectName": "Example Project",
    "event": {
        "type": "dashboard-list-secrets",
        "metadata": {
            "secretIds": [],
            "secretPath": "/",
            "environment": "staging",
            "numberOfSecrets": 0
        }
    },
    "actor": {
        "type": "user",
        "metadata": {
            "email": "morelmiles@gmail.com",
            "userId": "165ecdb4-406c-4c6d-8ca6-4dcb2f9603a9",
            "username": "morelmiles@gmail.com",
            "permission": {
                "metadata": {}
            }
        }
    }
}


- List, delete and revoke and create API keys 
- Make the integration in both backend and frontend 
