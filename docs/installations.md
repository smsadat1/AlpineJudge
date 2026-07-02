# Setup Guidelines

## Installating binary

``` bash
wget https://github.com/smsadat/alpinejudge/releases/download/v0.1.0/alpinejudge-linux-amd64.tar.gz
tar -xzf alpinejudge-linux-amd64.tar.gz
cd alpinejudge-linux-amd64
chmod +x setup.sh
```

## Setup Dispatcher
Look into `dispatcher/config.example.yaml` & `dispatcher/dispatcher.example.env` to create configuration files
``` bash
sudo ./setup.sh dispatcher --env /path/to/.env --config /path/to/config.yaml
```

## Setup Runner
Look into `runner/config.example.yaml` & `runner/runner.example.env` to create configuration files
``` bash 
sudo ./setup.sh runner --id runner-001 --env /path/to/.env
```


