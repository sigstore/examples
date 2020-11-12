# Rekor monitors

This repository contains various scripts to gather materials from various software release sites
and serialize the signing content into the rekor json format. Typically run as cron jobs or using
sensible timers, they will retrieve new release signing material when available and automatically
update rekors transparency log.