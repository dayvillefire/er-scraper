# ER-SCRAPER

SaaS migration tool from Emergency Reporting, as their APIs are woefully inadequate for data retention on migration (IMHO).

DISCLAIMER: This tool is meant to allow data migration from an obsolete service for fire service personnel who already have legal access to that information. No part of this software was designed to circumvent controls, but rather to automate a potentially tedious process.

## Quick How-to

Create a .env file with your username and password for Emergency Reporting in it:

```
USERNAME='chief'
PASSWORD='LuxuriousMustache123$$$'
```

Then, execute the `er-scraper` binary with the particular task you'd like it to run for exporting. This will dump out the data to the local path.

## Export Supports

- [X] Events / Calendar
- [X] Hydrants
- [X] Incidents (through NFIRS export, not necessary)
  - [ ] Incident Attachments
  - [ ] Incident Vehicles
- [ ] Occupancies
- [X] Training
  - [X] Training Files
- [ ] Users
  - [ ] User Certifications (and Certificates)
