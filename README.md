vagrantshadow 
=============

[![Build Status](https://drone.io/github.com/BenPhegan/vagrantshadow/status.png)](https://drone.io/github.com/BenPhegan/vagrantshadow/latest)

vagrantshadow provides a _very_ stripped back Vagrant Cloud capability that you can host behind a firewall with private boxes.  Currently it really only provides the discovery and download capability, and relies on `.box` files to be stored in a directory for discovery/serving.

It is a very rough first version, but it _appears_ to work.  Steps you need to undertake:

1. Copy the boxes that you want to serve into a directory (it is easier if this is the directory you will launch vagrantshadow from).
1. Ensure the boxes are named in the form `username-VAGRANTSLASH-boxname.box`.  These are the only ones that will get served.  They will default to v1.0 at the moment.  The provider will be checked and served correctly.
1. Run vagrantshadow.  It will default to a hostname of "localhost" and port of "8099".  These are important as they are where the boxes will be served from, so if you are hosting other than locally you will need to change this.

You should now have a hosted Vagrant Cloud!  To access this, you will need to do the following:

1. For Linux/Mac, type the following at a shell prompt: `export VAGRANT_SERVER_URL=http://localhost:8099` (adjust according to host/port).  This will redirect Vagrant to your server rather than Vagrant Cloud.
1. Use commands as per normal.  So far I have only tested the basics like `vagrant box add` and `vagrant init username/boxname`, and of course `vagrant up`.  So far so good.

Any issues, let me know!
