# sms-alert-system-cloud-formation

Cloud formation project for the Colectiva SMS alert system.

# Overview

The tool is used to:
1. Create a hosted zone with a domain name
1. Create an SSL certificate for the domain name
1. Create the service infrastructure as well as the pipeline from deploying the code to the infrastructure

Multiple stages are required because CloudFormation does not permit the SSL certificate to be verified using DNS
verification - only email verification. This creates a dependency loop because the email is sent to
`admin@${DOMAIN_NAME}.com`. The instance will not be created until the SSL certificate is validated and the SSL
cert can only be validated with an instance (running an email server).

Thus the decision was made to have two CloudFormation templates - one which controls the domain name set up and one
which does everything else. This enables us to still use DNS verification of the domain name in Route 53.

The architecture of the service is as follows (some minor details are left out):

<p align="center">
  <img src="docs/aws_quick_ref_arch.jpg" width="80%" height="80%"/>
</p>

A picture of the pipeline is:

**PIPELINE DIAGRAM HERE (NOT IMPLEMENTED YET)**

# Set Up

## Install Go

Install Go by following the instructions at https://golang.org/doc/install. If you are running Linux, you can install
it using your package manager. Remember to update your `PATH` to include the go commands:
```
PATH="~/go/bin:$PATH"
```

## Install Govendor

Go does not by default come with a package dependency system. However, there is a tool called `govendor`, which does
this. This tool was chosen over `godep` because it permits multiple versions of a given library to be installed.
Read more about the tool at https://github.com/kardianos/govendor.
```
# install govendor
go get -u github.com/kardianos/govendor
```

## Pull The Code and Install Dependencies

This will pull the code and the dependencies, putting the dependencies in `${PROJECT_ROOT}/vendor/`
```
govendor get github.com/ErrorsAndGlitches/wordpress-cloud-formation
```
The dependencies are defined in `./vendor/vendor.json`.

# Building

This will produce the executable with the same name as the package (`sms-alert-system-cloud-formation`):
```
govendor build
```

# Running

## Create the Domain Name and SSL Certificate

### Request a Domain Name

Details:
* This **MUST** use the **us-east-1** region. It is like S3 where the service is global and thus centered in the US
  Standard region, which is **us-east-1**.
* It is **HIGHLY** recommend to use a **.org** domain name so that the information associated with the domain name is
  hidden as detailed on the [Route 53 Domains .org webpage][].
* See [Route 53 Domains][] for more information on contact types
* Defaults to the **US** country code. If this needs to be used outside the US, please create an issue to add the
  support.
* Note the period in the phone number option.
* The zip code requites the 4 digit extension.
* Registering a domain name will automatically create a Route 53 Hosted Zone

This script uses the default contact type. Use the `--help` function to learn more about the CLI parameters.
```
./sms-alert-system-cloud-formation -p colectiva -r us-east-1 -s Gamma \
  register-domain-name \
  -d waisn-alert-sys-gamma.org \
  -f first_name -l last_name -e your.email@gmail.com \
  -o WAISN -t "201 SW 153rd St." -y Burien -x WA -z "98166-2313" -n +1.2069311514
```
This will print the operation id.

**You will recieve an email, which will require you to verify your email address. Failure to do so will cause the domain
to be susended in about 2 weeks.**

It can take a few minutes for the domain name registration to succeed. You can print the status via:
```
./sms-alert-system-cloud-formation -p colectiva -r us-east-1 -s Gamma print-record-status -i 'operation-id'
```

[Route 53 Domains .org webpage]: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/registrar-tld-list.html#org
[Route 53 Domains]: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/registrar-tld-list.html

### Print the Hosted Zone

The Hosted Zone ID, which was created when the domain name was registered, is required as a parameter for setting up the
SSL certificate.
```
./sms-alert-system-cloud-formation -p colectiva -s Gamma -r us-east-1 describe-hosted-zone -d waisn-alert-sys-gamma.org
```

### Setup the SSL Certificate

Use the Hosted Zone ID from the previous step to setup SSL. The default region can be used here.
```
./sms-alert-system-cloud-formation -p colectiva -s Gamma setup-ssl -d waisn-alert-sys-gamma.org -z "/hostedzone/00000000000000"
```

It can take up to 30 minutes for the certificate to be validated. You can run this command to see its current status:
```
./sms-alert-system-cloud-formation -p colectiva describe-ssl --ssl-arn "arn:aws:acm:us-west-2:000000000000:certificate/00000000-0000-0000-0000-000000000000"
```

## Create the Service Stack

### Create the CloudFormation Stack

This command will create the CloudFormation stack. It may take some time for all of the resources in the stack to be
procured. One of the arguments is a secret that the service uses, which you can read more about on the
[Play Framework Application Secret Documentation][]
```
./sms-alert-system-cloud-formation -p colectiva -s Gamma cf-service create \
  -d waisn-alert-sys-gamma.org \
  -b db_password \
  -a "arn:aws:acm:us-west-2:000000000000:certificate/00000000-0000-0000-0000-000000000000" \
  -t twilio_user \
  -w twilio_password \
  -n twilio_phone \
  -f play_framework_secret_key
```

[Play Framework Application Secret Documentation]: https://www.playframework.com/documentation/2.6.x/ApplicationSecret

##r Print the Elastic Load Balancer Public Domain Name

The output of the template is the load balancer's public domain name. After the stack has been created, it can be
printed with:
```
./sms-alert-system-cloud-formation -p colectiva -s Gamma cf-service describe
```

### Alias the Elastic Load Balancer

The final step is to forward requests sent to the domain name to the Elastic Load Balancer. This is done by creating an
**Alias** Record Set entry in the Hosted Zone. To do this, use the ELB's public domain name retrieved in the previous
step. Additionally, you will need to look up the hosted zone for the ELB based on the region that you created the ELB.
Note that the ELB created in the stack is an application ELB. The information can be found on the [ELB Region][] page.
```
./sms-alert-system-cloud-formation -p colectiva -s Gamma create-elb-alias \
  -d waisn-alert-sys-gamma.org \
  -z Z0000000000000 \
  -e AlertSysLoadBalancerGamma-0000000000.us-west-2.elb.amazonaws.com \
  -n Z111111111111
```
The Record Set should update instantaneously.

[ELB Region]: https://docs.aws.amazon.com/general/latest/gr/rande.html#elb_region

## Integrating with Twilio

The domain name should be used as the Twilio end-point.

# Contributing

Contributing to a Go projects takes a few extra steps compared to other languages. This is because the import statements
reference the actual github repository. Golang imports based on the file path from the `GOPATH` location. Thus we can
create a directory like the source repository, but then clone the fork in the directory:
```
USER_NAME=${GITHUB_USER_NAME}
PROJ_DIR="${GOPATH}/src/github.com/ColectivaLegal/"

mkdir -p "${PROJ_DIR}"
cd "${PROJ_DIR}"
# fetch the code
git clone git@github.com:${USER_NAME}/sms-alert-system-cloud-formation.git
# pull the dependencies into ./vendor/
govendor sync
```

Debug logging can be turned on by setting an environment variable:
```
export DEBUG=1
```

## Gotchas

* Note that the **Registrant Contact** in Route 53 Domains is also known as the **Bill Contact**.

# References

* [A reference architecture for deploying containerized microservices with Amazon ECS and AWS CloudFormation](https://github.com/awslabs/ecs-refarch-cloudformation)

