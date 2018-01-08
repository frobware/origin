// Package imagequalify provides functions for qualifying bare image
// references with a domain component based on a set of pattern
// matching rules. It also provides functions for parsing rules from
// files. A rules files is a line-orientated set of path-based
// patterns with an associated domain for each pattern.
//
// Each line is a sequence of two space-separated words; the first
// word is the image pattern, the second is a domain name. The comment
// character is hash (#), and comments run until the newline. Blank
// lines are ignored.
//
// Examples:
//
//   # Simple image references
//   busybox my-registry.com
//
//   # Images that reference a repository
//   library/busybox busybox.com
//
//   # Wildcards are supported using '*'
//
//   # any image in the openshift repository
//   openshift/* access.registry.redhat.com
//
//   */nginx     nginx.com  # nginx in any repository
//   */*:latest  dev.com    # any repo and any image with tag 'latest'
//   */*         foo.com    # any repo and any image
//   *           foo.com    # any image name
//
//   # Patterns can also reference image SHA's.
//   */*/*@sha256:<...SHA...> production.com
//
// Rule Ordering
//
// Rules are automatically ordered by the pattern's path; deeper paths
// match first and wilcards least (due to the collating sequence).
//
// Given the previous example the natural sorting order is:
//
//   openshift/*            access.registry.redhat.com
//   library/busybox        busybox.com
//   busybox                busybox.com
//   */nginx                nginx.com
//   */*:latest             dev.com
//   */*/*@sha256:<SHA>     production.com
//   */*                    foo.com
//   *                      foo.com
package imagequalify
