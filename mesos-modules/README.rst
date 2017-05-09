============================================
compose hook - cleanup dangling containers
============================================


Motivation
----------

We wish to cleanup after ourselves in case of executor failures. the leaked containers can lead to wasted resources on the slave node.
when configured this hook gets called when excutor exits. The hook cleansup all the currently running containers associated with exited executor. 
This module is also responsible for cleaning up any cgroups that are left behind by mesos slave. 

The module does not remove containers but only stops them but this behavior can be changed easily.


Goals
-----

This is a `Mesos compose module`_ which can be loaded using the enclosed
``modules.json`` descriptor

License & Allowed Use
---------------------

Top level licencing for project follows.


Build
-----

Prerequisites
^^^^^^^^^^^^^

You obviously need `Apache Mesos`_ to build this
project: in particular, you will need both the includes (``mesos``, ``stout``
and ``libprocess``) and the shared ``libmesos.so`` library.

In addition, Mesos needs access to ``picojson.h`` and a subset of the ``boost``
header files: see the
`3rdparty <https://github.com/apache/mesos/tree/master/3rdparty/libprocess/3rdparty>`_
folder in the mirrored github repository for Mesos, and in particular the
`boost-1.53.0.tar.gz <https://github.com/apache/mesos/blob/master/3rdparty/libprocess/3rdparty/boost-1.53.0.tar.gz>`_
archive.

The "easiest" way to obtain all the prerequisites would probably be to clone the Mesos
repository, build mesos and then install it in a local folder that you will then need to
configure.

In order to ease setting up  prerequisites and build the module you can you the provided dockerfile to build the module from desired version of mesos.
``docker build -t mesos-build -f Dockerfile-mesos-modules --build-arg MESOS_VERSION=1.1.0 .``
The built binary module can be fetched at ``/output`` by running the generated docker image. 

Usage
-----

In order to run a Mesos Agent with this module loaded, is a simple matter of
adding the ``--modules`` flag, pointing it to the generated JSON
``modules.json`` file. and enable hook using ``--hook`` option 

  $ ${MESOS_ROOT}/build/bin/mesos-slave.sh --work_dir=/var/lib/mesos \
      --modules=/path/to/execute-module/gen/modules.json \
      --hooks=org_apache_mesos_ComposePodCleanupHook \
      --master=zk://zkclusteraddress:port

See ``Configuration``  on the `Apache Mesos`_ documentation pages for more
details on the various flags.

Also, my `zk_mesos`_ github project provides an example `Vagrant`_
configuration showing how to deploy and run Mesos from the Mesosphere binary
distributions.


Tests
-----

Not yet implemented

--------
