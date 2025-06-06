# Gateway for Minecraft Servers

A hobby project made specifically for my own use case:
starting a Minecraft server instance on AWS when players try to join and
transferring them once the main server is up and running.

The protocol package abstracts the minecraft protocol in low-level.
Heavily influenced by [Tnze/go-mc](https://github.com/Tnze/go-mc).
It aims to cover pre-ingame stages and limited parts of ingame stage.
It only supports Minecraft 1.21.1 for now.

> **Disclaimer**: This codebase hasn't undergone any serious security audit.
Use it at your own risk.

## Setup

The gateway server would run 24/7 in a low-cost machine,
acting as an entry point that transfers whitelisted players to main servers.
It also spins up the main server when it's needed,
allowing it to just shutdown when idle and keep the cloud cost low.

### Benefits

* Reduces cloud service costs by letting the heavy main server sleep when idle
* Avoids Elastic IP usage
* Keeps the main server's IP hidden from strangers

### Configuration
The program looks for config.yaml and env variables (including .env file) for configuration.

Configurable settings are:
* `instance_id` (*INSTANCE_ID*): required, EC2 instance id of the main server
* `whitelist` (*WHITELIST*): optional, lists of whitelisted uuids
* `port`: optional, defaults to 25565, port of the main server

You might want to add these AWS configurations to env variables.
* *AWS_ACCESS_KEY_ID*
* *AWS_SECRET_ACCESS_KEY*
* *AWS_REGION*


## Long-term Plans (if i continue working on this)

* Support multiple minecraft versions
* Implement encrypted login
* Make the behavior highly customizable?
    * could be a lightweight player queue server
    * or a loadbalancer
