import sys

try:
    print(name)
except NameError:
    print("""Traceback (most recent call last):\n  File "./test_programs/complex2.py", line 1, in <module>
    print(name)
          ^^^^
NameError: name 'name' is not defined""", file=sys.stderr)

print("\n", end=sys.stderr)

try:
    a = 3
    print(a / 0)
except ZeroDivisionError:
    print("""Traceback (most recent call last):\n  File "./test_programs/complex2.py", line 15, in <module>
    print(a / 0)
          ~~^~~
ZeroDivisionError: division by zero""", file=sys.stderr)

exit(1)
