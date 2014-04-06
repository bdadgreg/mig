===================================================
Mozilla InvestiGator Concepts & Internal Components
===================================================
:Author: Julien Vehent <jvehent@mozilla.com>

.. sectnum::
.. contents:: Table of Contents

MIG is a platform to perform remote forensic on endpoints. It is composed of:

* Agent: a program that runs on endpoints and receives commands to run locally
  from the MIG platform. Commands are ran by agents using modules, such as
  'filechecker'.
* Scheduler: a router and processor that receives orders and forward them to
  agents
* API: an interface to the MIG platform used by investigators
* Clients: clients are used by investigators to interact with the MIG platform
  via the API
* Queue: a message queueing daemon that passes messages between the scheduler
  and the agents
* Database: a storage backend used by the scheduler and the api

Below is a high-level view of the different components:

 ::

    ( )               signed actions
    \|/  +------+  -----------------------> +-------+
     |   |client|    responses              | A P I |
    / \  +------+ <-----------------------  +-----+-+       +--------+
    investigator                                  +-------->|  data  |
                                                            |        |
                                              action/command|--------|
                                                            |        |
                                                  +-------->|  base  |
                                                  |         |        |
                      signed commands     +-------+---+     +--------+
                                          |           |
                      +++++--------------+| SCHEDULER |
                      |||||               |           |
                      vvvvv               +-----------+
                    +-------+                  ^^^^^
                    |       |                  |||||
                    |message|+-----------------+++++
                    |-------|     command responses
                    |broker |
                    |       |
                    +-------+
                    ^^    ^ ^
                    ||    | |
       +------------+|    | +-----------------+
       |           +-+    +--+                |
       |           |         |                |
    +--+--+     +--+--+    +-+---+          +-+---+
    |agent|     |agent|    |agent|  .....   |agent|
    +-----+     +-----+    +-----+          +-----+

Actions and Commands are messages passed between the differents components.

Actions and Commands
--------------------

Actions
~~~~~~~

Actions are JSON files created by investigator to perform tasks on agents.

For example, an investigator who wants to verify than root passwords are hashed
and salted on linux systems, would use the following action:

.. code:: json

	{
		"name": "Compliance check for Auditd",
		"description": {
			"author": "Julien Vehent",
			"email": "ulfr@mozilla.com",
			"url": "https://some_example_url/with_details",
			"revision": 201402071200
		},
		"target": "linux",
		"threat": {
			"level": "info",
			"family": "compliance"
		},
		"operations": [
			{
				"module": "filechecker",
				"parameters": {
					"/etc/shadow": {
						"regex": {
							"root password strongly hashed and salted": [
								"root:\\$(2a|5|6)\\$"
							]
						}
					}
				}
			}
		],
		"syntaxversion": 1
	}

The parameters are:

* Name: a string that represents the action.
* Target: a search string that will be used by the scheduler to find the agents
  the action will run on.
* Description and Threat: additional fields to describe the action
* Operations: an array of operations, each operation calls a module with a set
  of parameters. The parameters syntax are specific to the module.
* SyntaxVersion: indicator of the action format used. Should be set to 1.

Upon generation, additional fields are appended to the action:

* PGPSignature: all of the parameters above are concatenated into a string and
  signed with the investigators private GPG key. The signature is part of the
  action, and used by agents to verify that an action comes from a trusted
  investigator. `PGPSignature` is an array that contains one or more signature
  from authorized investigators. 
* ValidFrom and ExpireAt: two dates that constrains the validity of the action
  to a UTC time window.

Actions files are submitted to the API or the Scheduler directly. Eventually,
the PGP signature will be verified by intermediary components, and in any case
by each agent before execution.
Additional attributes are added to the action by the scheduler. Those are
defined as "MetaAction" and are used to track the action status.

Commands
~~~~~~~~

Upon processing of an Action, the scheduler will retrieve a list of agents to
send the action to. One action is then derived into Commands. A command contains an
action plus additional parameters that are specific to the target agent, such as
command processing timestamps, name of the agent queue on the message broker,
Action and Command unique IDs, status and results of the command. Below is an
example of the previous action ran against the agent named
'myserver1234.test.example.net'.

.. code:: json

	{
		"action":        { ... signed copy of action ... }
		"agentname":     "myserver1234.test.example.net",
		"agentqueueloc": "linux.myserver1234.test.example.net.55tjippis7s4t",
		"finishtime":    "2014-02-10T15:28:34.687949847Z",
		"id":            5978792535962156489,
		"results": [
			{
				"elements": {
					"/etc/shadow": {
						"regex": {
							"root password strongly hashed and salted": {
								"root:\\$(2a|5|6)\\$": {
									"Filecount": 1,
									"Files": {},
									"Matchcount": 0
								}
							}
						}
					}
				},
				"extra": {
					"statistics": {
						"checkcount": 1,
						"checksmatch": 0,
						"exectime": "183.237us",
						"filescount": 1,
						"openfailed": 0,
						"totalhits": 0,
						"uniquefiles": 0
					}
				},
				"foundanything": false
			}
		],
		"starttime": "2014-02-10T15:28:34.118926659Z",
		"status": "succeeded"
	}


The results of the command show that the file '/etc/shadow' has not matched,
and thus "FoundAnything" returned "false.
While the result is negative, the command itself has succeeded. Had a failure
happened on the agent, the scheduler would have been notified and the status
would be one of "failed", "timeout" or "cancelled".

Access Control Lists
--------------------

Not all keys can perform all actions. The scheduler, for example, sometimes need
to issue specific actions to agents (such as during the upgrade protocol) but
shouldn't be able to perform more dangerous actions. This is enforced by
Access Control Lists, or ACLs, stored on the agents. An ACL describes who can
access what function of which module. It can be used to require multiple
signatures on specific actions, and limit the list of investigators allowed to
perform an action.

ACLs are JSON documents that are currently hardwired into the agent, but will be
shipped dynamically to agents in the future (via an ACL module).

Below is an example of ACL for the `filechecker` module:

.. code:: json

	{
		"filechecker": {
			"requiredsignatures": 1,
			"authoritativesigners": [
				"E60892BB9BD89A69F759A1A0A3D652173B763E8F"
			]
		}
	}

`authoritativesigners` contains the PGP fingerprint of the public key of an
investigator. When an agent receives an action that calls the filechecker
module, it will first verify the signature of the action, and then validates
that the signer is authorized to perform the action.

The global ACL `all` can be used as a default for all modules. It has the
following syntax:

.. code:: json

	{
		"all": {
			"requiredsignatures": 1,
			"authoritativesigners": [
				"E60892BB9BD...",
				"9F759A1A0A3...",
				"A69F759A1A0..."
			]
		}
	}

The `all` ACL is overridden by module specific ACLs.

If a module requires multiple signatures, the `nonauthoritativesigners`
attribute can be used to list investigators that can sign, but which signature
isn't sufficient to launch the action. In addition, the attribute
`requiredauthoritativesigners` controls how many signatures from
`authoritativesigners` are required. If `requiredauthoritativesigners` is set to
0, and `requiredsignatures` is set to 2, then two `nonauthoritativesigners` can
sign and launch an action on this module without the approval of an
`authoritativesigners`.

.. code:: json

   {
		"firewall": {
			"requiredsignatures": 2,
			"requiredauthoritativesigners": 0
			"authoritativesigners": [
				"E60892BB9BD...",
				"9F759A1A0A3...",
				"A69F759A1A0..."
			],
			"nonauthoritativesigners": [
				"2FC05413E11...",
				"8AD5956347F..."
			}
		}
	}

ACL are currently applied to modules. In the future, ACLs should have finer
control to authorize access to specific functions of a module. For example, an
investigator could be authorized to call the `regex` function of filechecker
module, but only in `/etc`.

Agent registration process
--------------------------

Agent upgrade process
---------------------

Agent command execution flow
----------------------------

An agent receives a command from the scheduler on its personal AMQP queue (1).
It parses the command (2) and extracts all of the operations to perform.
Operations are passed to modules and executed asynchronously (3). Rather than
maintaining a state of the running command, the agent create a goroutine and a
channel tasked with receiving the results from the modules. Each modules
published its results inside that channel (4). The result parsing goroutine
receives them, and when it has received all of them, builds a response (5)
that is sent back to the scheduler(6).

When the agent is done running the command, both the channel and the goroutine
are destroyed.

 ::

             +-------+   [ - - - - - - A G E N T - - - - - - - - - - - - ]
             |command|+---->(listener)
             +-------+          |(2)
               ^                V
               |(1)         (parser)
               |               +       [ m o d u l e s ]
    +-----+    |            (3)|----------> op1 +----------------+
    |SCHED|+---+               |------------> op2 +--------------|
    | ULER|<---+               |--------------> op3 +------------|
    +-----+    |               +----------------> op4 +----------+
               |                                                 V(4)
               |(6)                                         (receiver)
               |                                                 |
               |                                                 V(5)
               +                                             (sender)
             +-------+                                           /
             |results|<-----------------------------------------'
             +-------+
