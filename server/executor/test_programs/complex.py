import sys

try:
    print(name)
except NameError:
    print("""Traceback (most recent call last):\n  File "./test_programs/complex.py", line 1, in <module>
    print(name)
          ^^^^
NameError: name 'name' is not defined""", file=sys.stderr)

try:
    a = 3
    print(a / 0)
except ZeroDivisionError:
    print("""Traceback (most recent call last):\n  File "./test_programs/complex.py", line 6, in <module>
    print(a / 0)
          ~~^~~
ZeroDivisionError: division by zero""", file=sys.stderr)

exit(1)
