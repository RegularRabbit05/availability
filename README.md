# Availability
Service to automatically execute tasks based on network availability. <br> <br>
This service has been created to fix a problem with Cloudflare WARP networking, which sometimes routes traffic through an unavailable (or non routable) network interface (such as the 169.254.169.254 endpoint), causing connectivity issues. <br>
The solution is to monitor network availability and restart the WARP connection to fix the issue. <br>