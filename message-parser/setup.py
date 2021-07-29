from setuptools import setup

setup(
   name='message-parser',
   version='1.0',
   description='Parser Redfish message using Message Registry',
   author='Jack Ding',
   author_email='jacding@redhat.com',
   packages=['message-parser'],
   install_requires=['python3-devel', 'gcc-c++'],
   scripts=[
            'scripts/cool',
            'scripts/skype',
           ]
)
