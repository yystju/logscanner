A simple log scanner to fetch the useful log information quickly.

Usage:
```
Usage of ./logscanner:
  -d string
    	The log folder to parse. (default "./logs/")
  -f string
    	The file name filter.
  -fr
    	The flag which represents whether the file filter is regular expression or not.
  -g string
    	The log file. The default "-" indicates that the executable name will be used as log name. (default "-")
  -l value
    	The line filters. This flag can be repeated for multiple times with different filter contents.
  -lr
    	The flag which represents whether the line filter is regular expression or not.
  -o string
    	The output file. default "-" indicates the standard output stream (stdout). (default "-")
```
