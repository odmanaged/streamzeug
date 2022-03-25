# SPDX-FileCopyrightText: Streamzeug Copyright © 2021 ODMedia B.V. All right reserved.
# SPDX-FileContributor: Author: Gijs Peskens <gijs@peskens.net>
# SPDX-License-Identifier: GPL-3.0-or-later


#This file is one giant hack, totally unsupported by upstream meson, 
#but for now, for our usecase (building with static dependencies) it works.

project('streamzeug','cpp', version: '0.3.2', meson_version: '>=0.57')

cmake = import('cmake')

srt_opt = cmake.subproject_options()
srt_opt.add_cmake_defines({'ENABLE_STDCXX_SYNC': true})
srt_opt.add_cmake_defines({'ENABLE_APPS': false})
srt_opt.add_cmake_defines({'ENABLE_SHARED': false})
srt_opt.add_cmake_defines({'ENABLE_STATIC': true})
srt_opt.set_install(false)

srt = cmake.subproject('libsrt', options: srt_opt)

cmlibsrt = srt.target('srt_static')

#SRT Target doesn't doesn't get build with the correct filename :/ 
#So it won't be found by Golang linker, let's copy it to the right location. 
mv = find_program('cp')
libsrt = custom_target(
    'libsrt',
    output: 'libsrt.a',
    input: cmlibsrt,
    command: [mv, '@INPUT@', '@OUTPUT@']
)

librist_proj = subproject(
    'librist',
    default_options: [
        'test=false',
        'default_library=static',
        'built_tools=false',
    ],
)

rist = dependency('librist', fallback: ['librist', 'librist_dep'])

sysconfdir = join_paths('/', get_option('sysconfdir'))
bindir = get_option('bindir')


conf_data = configuration_data()
conf_data.set('prefix', get_option('prefix'))
conf_data.set('bindir', bindir)
conf_data.set('sysconfdir', sysconfdir)

systemd = dependency('systemd', required: false)
if systemd.found()
    systemd_system_unit_dir = systemd.get_variable('systemdsystemunitdir')
    configure_file(
        input: 'streamzeug.service.in',
        install: true,
        install_dir: systemd_system_unit_dir,
        output: 'streamzeug.service',
	configuration: conf_data,
    )
endif

configure_file(
    copy: true,
    input: 'example.yaml',
    install: true,
    install_dir: join_paths(sysconfdir, 'streamzeug'),
    output: 'config.yaml.example',
)

ccpp = meson.get_compiler('cpp')

threads = dependency('threads')

inc = include_directories('dektec/DTAPI/Include')
dtapi_link_obj = ''
if host_machine.cpu_family() == 'x86_64'
	dtapi_link_obj = files('dektec/DTAPI/Lib/GCC5.1_CXX11_ABI1/DTAPI64.o')
else
	dtapi_link_obj = files('dektec/DTAPI/Lib/GCC5.1_CXX11_ABI1/DTAPI.o')
endif

dl = ccpp.find_library('dl')

libdektecasi = static_library(
    'dektec',
    'dektec/asi.cpp',
    include_directories: inc,
    objects: [
        dtapi_link_obj
    ],
    dependencies: [
        threads,
        dl,
	rist,
    ],
)

golang = find_program('go')
streamzeug_path = join_paths(meson.current_source_dir(), 'cmd/streamzeug/')

#HACK THIS MIGHT BREAK WHENEVER!
#ldflags may break whenever upstream srt decides to depend on other things.
ldflags = '-lsrt -lpthread -lstdc++  -lm -lgcc_s -lgcc -lc -lgcc_s -lgcc -lcrypto -L'+meson.current_build_dir()
pkg_config_path = join_paths(meson.current_build_dir(), 'meson-uninstalled')
libsrt_include_path = join_paths(meson.current_source_dir(), 'include_srt')

#HACK
cflags = '-I'+libsrt_include_path

#GOLANG HACKY
streamzeug_env = environment()
streamzeug_env.set('CGO_LDFLAGS', ldflags)
streamzeug_env.set('CGO_CFLAGS', cflags)

#At least libRIST works via pkg-config (relatively clean)
streamzeug_env.set('PKG_CONFIG_PATH', pkg_config_path)

streamzeug_env.set('CGO_CFLAGS_ALLOW', '.*')
streamzeug_env.set('CGO_LDFLAGS_ALLOW', '.*')

#HACK To make meson depend on the actual go files automatically
r = run_command('streamzeug_files.sh', meson.current_source_dir(), build_always_stale: true)
streamzeug_files = r.stdout().strip().split()
streamzeug_files += 'go.mod'
streamzeug_files += 'go.sum'
streamzeug_build_path = join_paths(meson.current_source_dir(), 'cmd/streamzeug')

r = run_command(
    'git',
    '--git-dir', join_paths(meson.current_source_dir(), '.git'),
    '--work-tree', meson.current_source_dir(),
    'describe',
    '--tags',
    '--dirty',
    '--match', 'v?.*',
    '--always'
)
vcs = r.stdout().strip()

streamzeug_ver = '-X \'github.com/odmedia/streamzeug/version.ProjectVersion=v@0@\''.format(meson.project_version())
streamzeug_gitver = '-X  \'github.com/odmedia/streamzeug/version.GitVersion=v@0@\''.format(vcs)
streamzeug_ldflags = '-ldflags= @0@ @1@'.format(streamzeug_ver, streamzeug_gitver)

streamzeug = custom_target(
    'streamzeug',
    output: 'streamzeug',
    env: streamzeug_env,
    build_always_stale: true,
    input: streamzeug_files,
    #build -a unfortunately needed due to linking with C libraries that might change.
    command: [ golang, 'build', streamzeug_ldflags, '-a', '-o','@OUTPUT@', streamzeug_build_path],
    install: true,
    install_dir: 'bin',
    depends: [
        libdektecasi,
        libsrt,
    ],
    install: true,
)
