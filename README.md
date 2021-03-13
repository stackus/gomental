# gomental

Get an idea of the mental burden your code might have, or the mental context it might require. In other words how much "stuff" you might need to keep in mind or sift through when working on the code.

Right now, and maybe never, the tool makes no determination of what might be excessive. You'll have to decide that for yourself.

## Installation
    go get -u github.com/stackus/gomental

## Usage

    Displays details about the golang source at the given path
    
    Usage:
    gomental path [flags]
    gomental [command]
    
    Examples:
    gomental my_source_path -d 0 --with-tests
    
    Available Commands:
    help        Help about any command
    version     displays the version
    
    Flags:
    -d, --depth int     Display an entry for all directories "depth" directories deep (default 1)
    -h, --help          help for gomental
    --no-zero           Ignore golang source free directories
    -s, --skip strings  Directory names to skip. format: dir,dir
    --with-tests        Include test files


### Example
 Running `gomental ftgogo --no-zero` on the source of [FTGOGO](https://github.com/stackus/ftgogo) displays the following report.

    Path            Packages  Files  Lines  Global Vars  Constants  Interfaces  Structs  Other Types  Methods  Funcs
    /accounting     6         18     835    7            0          2           34       1            33       23
    /consumer       6         15     565    7            0          3           18       2            17       18
    /delivery       5         18     1108   7            15         4           30       7            35       19
    /kitchen        6         29     1409   8            12         4           59       4            58       31
    /order          7         41     2308   10           4          7           69       4            100      44
    /order-history  5         11     796    2            6          2           21       2            17       13
    /restaurant     6         11     541    4            3          3           14       3            13       12
    /serviceapis    7         24     664    0            19         0           56       2            53       20
    /shared-go      9         22     1726   17           6          3           24       6            44       34

### More detailed depth report

    gomental my_source_path -d 3

### Single total report

    domental my_source_path -d 0

## License
MIT
