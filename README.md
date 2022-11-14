

## FlightRadar24

* **Documentation**: https://feed.flightradar24.com/fr24feed-manual.pdf
* **Config File**:  `/etc/fr24feed.ini`
* **Metrics/Health**: Not implemented; could use `fr24feed-status`
* **Logs**: `/var/log/fr24feed/fr24feed.log`
* **Net Data Format**:
  * `sbs1tcp`, whatever that is.
  * `beast-tcp` - outbound connection to host.
* **Web Port**: 8754. host:8754/flights.json

# RadarBox

* **Documentation**: 
* **Config File**:  `/etc/rbfeeder.ini`
* **Metrics/Health**:
  * `/dev/shm/rbfeeder_aircraft.json`, exported by dump1090_exporter.
  * `/dev/shm/rbfeeder_stats.json`, currently unused.
  * `/dev/shm/rbfeeder_status.json`, currently unused.
* **Logs**: `/var/log/rbfeeder.log`
* **Net Data Format**:
    * `raw`.
    * `beast-tcp` - outbound connection to host.
* **Web Port**: None found.

## PiAware

* **Documentation**: https://uk.flightaware.com/adsb/piaware/install
* **Config File**:  `/etc/piaware.conf`
* **Metrics/Health**:
  * `/run/piaware/status.json`, currently unused.
  * `piaware-status`, currently unused.
* **Logs**: `/var/log/piaware.log`
* **Net Data Format**: `beast-tcp`
* **Web Port**: None?
