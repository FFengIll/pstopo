# pstopo
Generate topo of process, including process status, fd, port, etc.

# workflow
- take a snapshot of current system info (process, net connection). (using `psutil`)
- analyse to match target using options and arguments.
- output the `dot` file, including
  - process relationship (pid, cmdline, port)
  - connection info (listen port and host)
  - etc.

# cli
## snapshot
`pstopo` can take a snapshot for current system status, 
then we can get topo from it and never lost original info (or changed while restart and so on). 

## topo
Creating a topo can work with process name, port number or pid number.
All the matches will be output.

```sh
pstopo process_name :port_number pid
```

Furthermore, if the number is a name, use `-n` or `--name` for it.

# template
The `pstopo` use `dot` (aka `graphviz`) as default output, and then to svg / png / etc.

Using `dot`, pstopo will allow to customize output style for different information.

Template engine `` is used and work with some inline variable as bellow
- port
- pid
- cmdline

# TODO
- [ ] analyse information of system process and port
- [ ] search and match information
- [ ] build a topo graph of the match
- [ ] output topo graph using graphviz
- [ ] serialize and deserialize system information (as snapshot)
- [ ] support template
- [ ] support customize template (for some info only)

# license
[MIT](LICENSE).