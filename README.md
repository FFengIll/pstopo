# pstopo
Generate topo of process, including process status, port, ~~fd~~ and etc.

# workflow
- take a snapshot of current system info (process, net connection). (using `psutil`)
- analyse to match target using config (json) and arguments.
- output the `dot` file, including
  - process relationship (pid, cmdline, port)
  - connection info (listen port and host)
  - etc.

# usage
## topo
Creating a topo can work with process name, port number or pid number.
All the matches will be output.

```sh
# auto generate snapshot, and filter them to output the output.dot
# if no snapshot, pstopo will take one
pstopo process_name :port_number pid

# specific existed snapshot and existed topo (config)
pstopo -s snapshot.json -t topo.json

# dynamic add config itme
pstopo -s snapshot.json -t topo.json :8080 zsh
```

~~Furthermore, if the number is a name, use `-n` or `--name` for it.~~

## snapshot
`pstopo` can take a snapshot for current system status, 
then we can get topo from it and never lost original info (or changed while restart and so on). 

```sh
pstopo snapshot -o yourname.json
```


## template (WIP)
The `pstopo` use `dot` (aka `graphviz`) as default output, and then to svg / png / etc.

Using `dot`, pstopo will allow to customize output style for different information.

Template engine `text/template` is used and work with some inline variable as bellow
- port
- pid
- cmdline

# features
- [x] analyse information of system process and port
- [x] search and match information
- [x] build a topo graph of the match
- [x] output topo graph using graphviz
- [x] serialize and deserialize process information (as snapshot)
- [x] support template
- [ ] support customize template (for some info only)

# license
[MIT](LICENSE).