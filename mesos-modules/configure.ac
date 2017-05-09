AC_INIT([mesos-modules],[0.2])
AC_PREREQ([2.69])

AC_CANONICAL_HOST
AC_CANONICAL_BUILD
AC_CANONICAL_TARGET

AC_CONFIG_FILES([Makefile])
AC_LANG([C++])
AC_PROG_CC

# If the user did not provide any overrides of `CXXFLAGS` then assume
# he actually means to have them empty. This may seem surprising to
# experienced autotools users knowing that `AC_PROG_CXX` sets `CXXFLAGS`
# to a default (commonly `-g -O2`) but note that debugging symbols and
# optimizing are explicitely set further down using `MESOS_CXXFLAGS`.
old_CXXFLAGS="$CXXFLAGS"
AC_PROG_CXX
if test "x$old_CXXFLAGS" = "x"; then
    CXXFLAGS=
fi

AC_PROG_CC_C_O
AC_CONFIG_MACRO_DIR([m4])

AM_INIT_AUTOMAKE([foreign])

LT_PREREQ([2.2])
LT_INIT
LT_LANG([C++])
LT_OUTPUT


###############################################################################
# Optional features.
###############################################################################

AC_ARG_ENABLE([debug],
              AS_HELP_STRING([--enable-debug],
                             [Enable debug symbols @<:@default=yes@:>@]),
              [], [enable_debug=yes])

AC_ARG_ENABLE([optimize],
              AS_HELP_STRING([--enable-optimize],
                             [Enable optimizing @<:@default=no@:>@]),
              [], [enable_optimize=yes])


###############################################################################
# Optional packages.
###############################################################################

AC_ARG_WITH([mesos],
            [AS_HELP_STRING([--with-mesos=@<:@=DIR@:>@],
                            [provide root dir for Mesos installation])],
            [mesos=$withval], [])

AC_ARG_WITH([mesos_root],
            [AS_HELP_STRING([--with-mesos-root=@<:@=DIR@:>@],
                            [provide root dir for Mesos sources])],
            [mesos_root=$withval], [])

AC_ARG_WITH([mesos_build_dir],
            [AS_HELP_STRING([--with-mesos-build-dir=@<:@=DIR@:>@],
                            [provide Mesos build dir])],
            [mesos_build_dir=$withval], [])

AC_ARG_WITH([protobuf],
            [AS_HELP_STRING([--with-protobuf=@<:@=DIR@:>@],
                            [provide install location of protobuf])],
            [with_protobuf=$withval], [])

AC_ARG_WITH([glog],
            [AS_HELP_STRING([--with-glog=@<:@=DIR@:>@],
                            [provide install location of glog])],
            [with_glog=$withval], [])

AC_ARG_WITH([openssl],
            [AS_HELP_STRING([--with-openssl=@<:@=DIR@:>@],
                            [provide install location of openssl])],
            [with_openssl=$withval], [])


MESOS_CXXFLAGS=

###############################################################################
# Debug/Optimization checks.
###############################################################################

debug_flags=""
if test "x$enable_debug" = "xyes"; then
  debug_flags="-g"
fi

optimize_flags="-O0"
if test "x$enable_optimize" = "xyes"; then
  optimize_flags="-O2"
fi

MESOS_CXXFLAGS="${MESOS_CXXFLAGS} $debug_flags $optimize_flags"

###############################################################################
###############################################################################
###############################################################################

MESOS_ROOT="$mesos_root"
AC_SUBST([MESOS_ROOT])

MESOS_BUILD_DIR="$mesos_build_dir"
AC_SUBST([MESOS_BUILD_DIR])

if test -d "$mesos"; then
  MESOS_CPPFLAGS="-I${mesos}/include"
  if test -d "${mesos}/lib64"; then
    MESOS_LDFLAGS="-L${mesos}/lib64 -lmesos"
  else
    MESOS_LDFLAGS="-L${mesos}/lib -lmesos"
  fi
else
  if test -d "$mesos_root"; then
    MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -I${mesos_root}/include"

    # Add src to include path.
    MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -I${mesos_root}/src"

    # Add libprocess to include path.
    MESOS_CPPFLAGS+=" -I${mesos_root}/3rdparty/libprocess/include"

    # Add stout to include path.
    MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -I${mesos_root}/3rdparty/stout/include"
  else
    AC_MSG_ERROR([Invalid mesos path; use --with-mesos-root=<DIR>
                  <DIR> is the top-level directory of the Mesos distribution.])
  fi

  if test -d "$mesos_build_dir"; then
    # Add built sources (i.e. protoc'd protobufs).
    MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -I${mesos_build_dir}/include"

    # add picojson and boost headers
    MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -I${mesos_build_dir}/3rdparty/boost-1.53.0"
    MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -I${mesos_build_dir}/3rdparty/picojson-1.3.0"

    # Add src to include path.
    MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -I${mesos_build_dir}/src"

    # Add the built libmesos.
    MESOS_LDFLAGS="${MESOS_LDFLAGS} -L${mesos_build_dir}/src -lmesos"
  else
    AC_MSG_ERROR([Invalid mesos build path; use --with-mesos-build-dir=<DIR>
                  <DIR> is the Mesos build directory.])
  fi
fi

if test -n "`echo $with_protobuf`"; then
  # Check for the protobuf compiler at user-provided location.
  AC_PATH_TOOL([PROTOCOMPILER], [protoc], [], [${with_protobuf}/bin])
  MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -I${with_protobuf}/include"
  MESOS_LDFLAGS="${MESOS_LDFLAGS} -L${with_protobuf}/lib -lprotobuf"
else
  # Check for the protobuf compiler in default PATH.
  AC_PATH_TOOL([PROTOCOMPILER], [protoc], [], [$PATH])
fi

if test -z "`echo $PROTOCOMPILER`"; then
  AC_MSG_ERROR([protoc not found in PATH])
fi

if test -n "`echo $with_glog`"; then
  MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -I${with_glog}/include"
  MESOS_LDFLAGS="${MESOS_LDFLAGS} -L${with_glog}/lib -lglog"
fi

if test -n "`echo $with_openssl`"; then
  MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -I${with_openssl}/include"
  MESOS_LDFLAGS="${MESOS_LDFLAGS} -L${with_openssl}/lib"
  MESOS_LDFLAGS="${MESOS_LDFLAGS} -lssl -lcrypto"
fi


# Enable flag to check whether root and build dir were provided.
AM_CONDITIONAL([SOURCE_PROVIDED], [test -d "$mesos_root"])

AC_SUBST(MESOS_CPPFLAGS)
AC_SUBST(MESOS_CXXFLAGS)
AC_SUBST(MESOS_LDFLAGS)

AC_MSG_NOTICE([Setting up build environment for ${target_cpu} ${target_os}])

# Determine the current OS.
case "${target_os}" in
  linux*)
    OS_NAME=linux
    LIB_EXT="so"
    ;;
  *)
    AC_MSG_ERROR("Modules are currently unsupported on your platform.")
esac

# Used for conditionally including source files.
AM_CONDITIONAL([OS_LINUX], [test "x$OS_NAME" = "xlinux"])

AC_SUBST(LIB_EXT)

# Check if we're using clang.
AC_MSG_CHECKING([if compiling with clang])

AC_LANG_PUSH([C++])
AC_COMPILE_IFELSE(
[AC_LANG_PROGRAM([], [[
#ifndef __clang__
       not clang
#endif
]])],
[CLANG=yes], [CLANG=no])
AC_LANG_POP([C++])

AC_MSG_RESULT([$CLANG])
AC_SUBST([CLANG])

if test "x$CLANG" = "xno"; then
  # Check the version of gcc and add any flags as appropriate. Note
  # that '-dumpversion' works for clang as well but as of clang 3.3 it
  # reports version 4.2.1 (for gcc backwards compatibility).
  GCC_VERSION="`${CC} -dumpversion`"
  AC_MSG_NOTICE([GCC version: $GCC_VERSION])
  test $? = 0 || AC_MSG_ERROR([failed to determine version of gcc])

  # Check for GCC version 4.4.
  AX_COMPARE_VERSION([$GCC_VERSION], [eq2], [4.4],
                     [is_gxx44=yes], [is_gxx44=no])
  if test "x$is_gxx44" = "xyes"; then
    AC_MSG_NOTICE([Setting up CXXFLAGS for g++-4.4])
    # We fail to build some protobuf generated code with gcc 4.4
    # without setting -fno-strict-aliasing.
    CFLAGS="$CFLAGS -fno-strict-aliasing"
    CXXFLAGS="$CXXFLAGS -fno-strict-aliasing"
  fi

  # Check for GCC version >= 4.8.
  AX_COMPARE_VERSION([$GCC_VERSION], [ge], [4.8],
                     [is_ge_gxx48=yes], [is_ge_gxx48=no])
  if test "x$is_ge_gxx48" = "xyes"; then
    AC_MSG_NOTICE([Setting up CXXFLAGS for g++ version >= 4.8])
    # Boost 1.53.0 fails to compile with GCC 4.8 without
    # -Wno-unused-local-typedefs, and automake does not recognize the
    # flag.
    # TODO(brenden): Remove this when Boost has a resolution.
    CFLAGS="${CFLAGS} -Wno-unused-local-typedefs"
    CXXFLAGS="${CXXFLAGS} -Wno-unused-local-typedefs"
  fi

  # Check for GCC version == 4.7 and fail. Starting with 4.7 the '#if
  # __cplusplus >= 201103L' will evaluate to true which means the
  # C++11 code paths will be compiled but certain C++11 features that
  # we use are not supported by 4.7. Since we're requiring C++11 going
  # forward we can't support this compiler.
  AX_COMPARE_VERSION([$GCC_VERSION], [eq2], [4.7],
                     [is_gxx47=yes], [is_gxx47=no])
  if test "x$is_gxx47" = "xyes"; then
    AC_MSG_ERROR([Unable to build with g++-4.7 due to missing C++11 features])
  fi

  # Suppress some erroneous warnings that break the build.
  MESOS_CPPFLAGS="${MESOS_CPPFLAGS} -Wno-maybe-uninitialized"
fi

# Ensure we can build the C++11 features we expect, and set the std
# CXXFLAG as appropriate.
AX_CXX_COMPILE_STDCXX_11([noext], [mandatory])

old_CPPFLAGS=${CPPFLAGS}
CPPFLAGS="${CPPFLAGS} ${MESOS_CPPFLAGS}"

old_LDFLAGS=${LDFLAGS}
LDFLAGS="${LDFLAGS} ${MESOS_LDFLAGS}"

# Check if headers and library were located.
AC_CHECK_HEADER([glog/logging.h],
                [],
                [AC_MSG_ERROR([cannot find glog header.])])

AC_CHECK_HEADER([google/protobuf/message.h],
                [],
                [AC_MSG_ERROR([google protobuf is not installed.])])

AC_CHECK_HEADERS([openssl/ssl.h],
                 [],
                 [AC_MSG_ERROR([cannot find libssl headers])])

AC_CHECK_HEADERS([boost/lexical_cast.hpp boost/functional/hash.hpp],
                [],
                [AC_MSG_ERROR([boost is not installed.])])

AC_CHECK_HEADER([picojson.h],
                [MESOS_CPPFLAGS+=" -DPICOJSON_USE_INT64 -D__STDC_FORMAT_MACROS"],
                [AC_MSG_ERROR([picojson is not installed.])])


LDFLAGS=${old_LDFLAGS}
CPPFLAGS=${old_CPPFLAGS}


# Create Modules JSON blobs.
AC_CONFIG_FILES([hook/modules.json], [])

AC_OUTPUT
