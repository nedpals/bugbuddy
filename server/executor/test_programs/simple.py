import sys

try:
    print(name)
except NameError:
    print("""Traceback (most recent call last):
  File "./test_programs/simple.py", line 4, in <module>
    print(name)
          ^^^^
NameError: name 'name' is not defined""", file=sys.stderr)
    exit(1)
