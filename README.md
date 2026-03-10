# CPM Anchor
This repository contains the source code, a built release (downloaded by the installation script), and installation script. I.e. all the components needed for the anchor executor that is used by the ColonyPM system to facilitate deployments.

## Installation
The prerequisites to integrate a node to the ColonyPM system are:
1. Docker
2. Five environment variables

The five environment variables are (with examples):
```bash
export HOST="colony.colonypm.xyz"  # The ColonyOS Server HOST
export PORT="443"                  # The ColonyOS Server PORT
export INSECURE="false"            # Wether the ColonyOS Server uses TLS or not
export EXE_NAME="my-anchor"        # The executor name this anchor will use (since the anchor is an executor)
export PRVKEY="..."                # A Private key the executor can use.
```
Once these prerequisites are met, all that's left to do is to run:
```bash
curl -sL executor.colonypm.xyz | bash
```
