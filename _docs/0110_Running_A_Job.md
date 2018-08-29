---
title: Runing a Job - the hard way
classes: wide
toc: true
---
## Hello World!
Our first job will be a "Hello World!" example using the Busybox image from
DockerHub.

### Create a job json file
Here is our Hello World job definition in json:
{% highlight json %}
{% include_relative scripts/hello_world/hello_world_job.json %}
{% endhighlight %}

### Running the job
To submit this there are a number of steps, see this script (this example
script is running against the Dev Vagrant instance of gostint and as such
is running Vault in dev mode, hence the login token of '`root`'):
{% highlight bash %}
{% include_relative scripts/hello_world/run.sh %}
{% endhighlight %}
