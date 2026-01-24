### Mutator


1. Initialize the Project
Create a new directory and run the init command. This generates the missing PROJECT file.

```sh
kubebuilder init --domain svelama.com --repo github.com/svelama/mutator
```

2. Create API (Resource and Controller)
Before we can create webhook, we need to create the resource it will act upon
```sh
kubebuilder create api --group ship --version v1 --kind Frigate --resource --controller
```

3. Create the Mutating Webhook

Now that the project and API exist, you can successfully run the webhook command.

```sh
kubebuilder create webhook --group ship --version v1 --kind Frigate --defaulting
```


Create K3D cluster and install cert manager

```sh
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.2/cert-manager.yaml
```

