# flavor-of-the-week
Discord bot that manages the "flavor of the week"

## How to deploy
1. [Link a repo in Cloud build](https://cloud.google.com/build/docs/automating-builds/github/connect-repo-github?generation=2nd-gen)
2. Create a bucket for your tfstate backend
3. Configure your tfvars and backend.conf
4. Deploy the pipeline
    ```
    $ cd flavor-of-the-week
    $ terraform init -backend-config=backend.conf
    $ terraform plan
    $ terraform apply
    ```