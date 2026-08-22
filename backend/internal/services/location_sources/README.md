# Bangladesh Location Source Data

Canonical runtime catalog:
- ../location_catalog.csv

Required runtime columns:
- division
- district
- upazila
- post_office
- postal_code

Source policy:
- Administrative hierarchy is reconciled against the existing approved
  TS-Cloud location master.
- Post office and postal code values are collected from official
  Bangladesh Post sources.
- Do not guess a missing postal code.
- If the official source leaves a postal code blank, keep it blank.
- Source/staging data must be validated before replacing the runtime catalog.
