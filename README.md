vagrantshadow 
=============

[![Build Status](https://drone.io/github.com/BenPhegan/vagrantshadow/status.png)](https://drone.io/github.com/BenPhegan/vagrantshadow/latest)

vagrantshadow provides a _very_ stripped back Vagrant Cloud capability that you can host behind a firewall with private boxes.  Currently it really only provides the discovery and download capability, and relies on `.box` files to be stored in a directory for discovery/serving.

Steps you need to undertake to use vagrantshadow:

1. Copy the boxes that you want to serve into a directory (it is easier if this is the directory you will launch vagrantshadow from).
1. Ensure the boxes are named in the form `username-VAGRANTSLASH-boxname__version_provider.box`.  These are the only ones that will get served.  If there are multiple versions per box, the highest version box will be set as current by default.
1. Run vagrantshadow.  It will default to a hostname of "localhost" and port of "8099".  These are important as they are where the boxes will be served from, so if you are hosting other than locally you will need to change this.  If you are exposing the service on a server with an external hostname of "acme.org" ensure that this is the value you pass to vagrantshadow, as this will be used to construct the download URLs.

You should now have a hosted Vagrant Cloud!  To access this, you will need to do the following:

1. For Linux/Mac, type the following at a shell prompt: `export VAGRANT_SERVER_URL=http://localhost:8099` (adjust according to host/port).  This will redirect Vagrant to your server rather than Vagrant Cloud.
1. Use commands as per normal.  Versions will be reported correctly, allowing version updates and alerts.

Any issues, let me know!
