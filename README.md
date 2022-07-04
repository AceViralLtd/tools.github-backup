# Github backup tool
This tool will scan the given organisation and archive all repos to aws glacier


## installation
- copy .ghb.example.toml to .ghb.toml
- fill out the config file with valid details
    - github user mush have full access to all repos in the org
    - aws user musth have write permissions on glacier
- build the project `make build`

## Usage
```sh
./github-backup -h
Archive github repos to aws glacier

Options:
    --date
    Overwrite the target date with the given one

    --config[=.ghb.toml]
    Path to the config file

    --help, -h[=false]
    Show this document
```
