import sys

print("Ooops I have been included in error", file=sys.stderr)

try:
    print(name)
except NameError:
    print("""Traceback (most recent call last):\n  File "./test_programs/dangling.py", line 1, in <module>
    print(name)
          ^^^^
NameError: name 'name' is not defined""", file=sys.stderr)

exit(1)
