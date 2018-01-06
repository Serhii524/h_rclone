package lshelp

// Help describes the common help for all the list commands
var Help = `
Any of the filtering options can be applied to this commmand.

There are several related list commands

  * ` + "`ls`" + ` to list size and path of objects only
  * ` + "`lsl`" + ` to list modification time, size and path of objects only
  * ` + "`lsd`" + ` to list directories only
  * ` + "`lsf`" + ` to list objects and directories in easy to parse format
  * ` + "`lsjson`" + ` to list objects and directories in JSON format

` + "`ls`,`lsl`,`lsd`" + ` are designed to be human readable.
` + "`lsf`" + ` is designed to be human and machine readable.
` + "`lsjson`" + ` is designed to be machine readable.

Note that ` + "`ls`,`lsl`,`lsd`" + ` all recurse by default - use "--max-depth 1" to stop the recursion.

The other list commands ` + "`lsf`,`lsjson`" + ` do not recurse by default - use "-R" to make them recurse.
`
