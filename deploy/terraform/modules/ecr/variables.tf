variable "name" { type = string }
variable "images" {
  type    = list(string)
  default = ["redirect", "api", "workers", "web"]
}
