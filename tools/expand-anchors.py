import sys
import yaml

# A Dumper to derefernce all of our anchors. https://ttl255.com/yaml-anchors-and-aliases-and-how-to-disable-them/
class NoAliasDumper(yaml.SafeDumper):
    def ignore_aliases(self, data):
        return True

yaml_location = "api.yaml"
ouput_location = "expanded.yaml"
if (len(sys.argv) > 1):
    yaml_location = sys.argv[1]

if (len(sys.argv) > 2):
    ouput_location = sys.argv[2]
in_yaml = open(yaml_location, "r")
# Note that the produced YAML has a bug with \n appearing in our lambda templates. These templates should not be
# consumed by the openapi generator so it doesn't matter, but should be watched out for if we use this for anything else
out_yaml = open(ouput_location, "w")
# Loader=yaml.FullLoader is actually the default behavior but it gives you a warning if you aren't explicit
# default_flow_style keeps everything in block style, the default is JSON grouping
yaml.dump(yaml.load(in_yaml, Loader=yaml.FullLoader), out_yaml, Dumper=NoAliasDumper, default_flow_style=False)
