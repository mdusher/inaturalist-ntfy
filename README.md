# Observation tracker for iNaturalist

This is a simple script that checks for new observations of specified taxon IDs in a specified place and sends a push notification using ntfy.sh.

No support is offered for this script, so feel free to modify as needed.

## Configuration

Configuration is done using environment variables.

| Name | Description |
| --- | --- |
| NTFY_URL | ntfy URL to send notifications to |
| NTFY_TOKEN | Token to authorise to the ntfy endpoint with |
| TAXON_IDS | Comma separated list of taxon IDs. No spaces |
| PLACE_ID | ID of place to search for observations in |

Example:
```
NTFY_URL="https://notify.example.local/inaturalist"
NTFY_TOKEN="tk_xxx555xxxxxxxx55xxx5xxxx5xxxx6xx"
TAXON_IDS="14539,14153" # 14539 = Satin Bowerbird, 14153 = Eastern Yellow Robin
PLACE_ID="7830" # 7830 = Victoria, Australia
```

