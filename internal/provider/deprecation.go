package provider

// deprecationMessage is attached to the provider and every resource and data
// source schema. Terraform shows it as a warning during plan/validate, and
// tfplugindocs marks the generated docs as deprecated.
const deprecationMessage = "The CloudEndure provider is deprecated and no longer maintained. " +
	"AWS has discontinued CloudEndure Disaster Recovery; migrate to AWS Elastic Disaster Recovery (AWS DRS)."

// deprecationNotice is prepended to the markdown descriptions rendered by
// tfplugindocs, which does not display schema deprecation messages itself.
const deprecationNotice = "!> **Deprecated:** " + deprecationMessage + "\n\n"
