#!/usr/bin/env perl
# Copyright (c) 2015 Yon <anaseto@bardinflor.perso.aquilenet.fr>
#
# Permission to use, copy, modify, and distribute this software for any
# purpose with or without fee is hereby granted, provided that the above
# copyright notice and this permission notice appear in all copies.
#
# THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
# WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
# MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
# ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
# WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
# ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
# OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

################################################################################
# This script generates colored html suitable for inclusion in <pre>. It reads
# from standard input and writes to stdout.
#
# Use it in a frundis file with:
#
#   .X ftag -t frundis-code -shell "perl html_coloration.pl"
#   .If -as-is -t frundis-code source.frundis
#
# Then you can parameter your css to color as you want. Available classes are:
#   "comment", "macro", "ppmacro", "escape"
# This regexp-based script does not parse the complete language, so you may
# want to tweek it and add new classes. For more control, you may prefer to
# write a Go script based on the parser package.
################################################################################

use utf8;
use strict;
use warnings;
use v5.12;
use open qw(:std :utf8);

my %escapes = (
    '&' => '&amp;',
    '"' => "&quot;",
    "'" => "&apos;",
    "<" => "&lt;",
    ">" => "&gt;",
);
my $cmd = 0;
my $longcmd = 0;

while (<>) {
    chomp;
    my $html;
    my $line = $_;
    $line =~ s/^\.\\".*//;
    $line =~ s/\\".*//;
    if    ($line =~ /^\./) { $cmd = 1; }
    elsif ($longcmd) { $cmd = 1; }
    else                            { $cmd = 0; }
    if   ($line =~ /^\..*\\$/) { $longcmd = 1; }
    else { $longcmd = 0 }
    if (/^\.\s*\\"/a) {
        $html = do_comment($_);
    }
    elsif (/(.*?)(\\".*)/) {
        my $code = $1;
        my $comment = $2;
        $comment = do_comment($comment);
        $code = do_code($code);
        $html = $code . $comment;
    }
    else {
        $html = do_code($_);
    }
    print $html, "\n";
}

sub do_code {
    my $text = shift;
    $text =~ s/("|<|>|'|&)/$escapes{$1}/ge;
    if ($text !~ /^\.(?:#|X)/) {
        $text =~ s|^(\.\s*\S+)|<span class="macro">$1</span>|a
    } else {
        $text =~ s|^(\.\s*\S+)|<span class="ppmacro">$1</span>|a;
    }
    $text =~ s#(
         \\&amp;
        |\\e
        |\\~
        |\\\$\d+
        |\\\$@
        |\\\*\[[^\]]*\]
        )#<span class="escape">$1</span>#xg;
    $text =~ s|(\\$)|<span class="escape">$1</span>|g if $cmd;
    $text =~ s{(\s)(-\S*)}{$1<span class="option">$2</span>}g;
    return $text;
}

sub do_comment {
    my $text = shift;
    $text =~ s/("|<|>|'|&)/$escapes{$1}/ge;
    $text = qq{<span class="comment">$text</span>};
    return $text;
}
