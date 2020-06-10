# portal_patterns
## What it does?
Find a *largest* pattern of a given type you can draw on given set of portals.

There is both command line version and a (simplistic) GUI version.
## What kinds of patterns does it support?
* homogeneous fields
* herringbone fields (one side or two side)
* cobweb pattern
* deepest fields from a given three sets of corner portals
* furthest drone flights
## How to get a list of portals?
Script now accepts only JSON or CSV files in format exported by **Multi Export IITC Plugin** (https://github.com/modkin/Ingress-IITC-Multi-Export)
## How to run the GUI version?
Install Tcl/Tk distribution, version at least 8.6.

On Linux you can install the the tk8.6 and tcl8.6 packages.

On Windows you can download ActiveTcl distribution from https://www.activestate.com/products/tcl/downloads/ - while installing make sure to add the TclTk path to the PATH evironment variable.
