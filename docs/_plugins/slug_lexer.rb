# frozen_string_literal: true

require "rouge"

module Rouge
  module Lexers
    class Slug < RegexLexer
      title "Slug"
      desc "Slug programming language"
      tag "slug"
      aliases "slug"
      filenames "*.slug"

      state :root do
        rule %r/\s+/m, Text::Whitespace
        rule %r/\/\/\/.*$/, Comment::Doc
        rule %r/\/\/.*$/, Comment::Single
        rule %r/#.*$/, Comment::Single

        rule %r/0x"(?:[0-9a-fA-F]{2})*"/, Num::Hex
        rule %r/0x(?:_?[0-9a-fA-F]{2})+/, Num::Hex
        rule %r/\b\d(?:[\d_]*\d)?(?:\.\d(?:[\d_]*\d)?)?[eE][+-]?\d(?:[\d_]*\d)?\b/, Num::Float
        rule %r/\b\d(?:[\d_]*\d)?\.\d(?:[\d_]*\d)?\b/, Num::Float
        rule %r/\b\d(?:[\d_]*\d)?\b/, Num::Integer

        rule %r/"""/, Str::Double, :triple_double
        rule %r/'''/, Str::Single, :triple_single
        rule %r/"/, Str::Double, :double
        rule %r/'/, Str::Single, :single

        rule %r/@[A-Za-z_][A-Za-z0-9_]*/, Name::Decorator

        rule %r/\b(true|false|nil)\b/, Keyword::Constant
        rule %r/\b(var|val)\b/, Keyword::Declaration
        rule %r/\b(fn|foreign|match|struct|copy)\b/, Keyword
        rule %r/\b(if|else)\b/, Keyword
        rule %r/\b(return|recur|throw|defer|onsuccess|onerror)\b/, Keyword
        rule %r/\b(nursery|limit|spawn|await|within)\b/, Keyword

        rule %r{/>|=>|\.\.\.|\?\?\?|:\+|\+:}, Operator
        rule %r/[=!<>]=?|&&|\|\||<<|>>|[+\-*\/%~^&|]/, Operator
        rule %r/[(){}\[\],.;:]/, Punctuation

        rule %r/[A-Za-z_][A-Za-z0-9_]*/, Name
      end

      state :double do
        rule %r/\\[\\"'nrt\{]/, Str::Escape
        rule %r/\\[0-7]{1,3}/, Str::Escape
        rule %r/\{\{/, Str::Interpol, :interpolation
        rule %r/[^"\\]+/, Str::Double
        rule %r/"/, Str::Double, :pop!
        rule %r/\\/, Str::Double
      end

      state :triple_double do
        rule %r/\\[\\"'nrt\{]/, Str::Escape
        rule %r/\\[0-7]{1,3}/, Str::Escape
        rule %r/\{\{/, Str::Interpol, :interpolation
        rule %r/"""/, Str::Double, :pop!
        rule %r/[^\\"]+/, Str::Double
        rule %r/"/, Str::Double
        rule %r/\\/, Str::Double
      end

      state :single do
        rule %r/[^']+/, Str::Single
        rule %r/'/, Str::Single, :pop!
      end

      state :triple_single do
        rule %r/[^']+/, Str::Single
        rule %r/'''/, Str::Single, :pop!
        rule %r/'/, Str::Single
      end

      state :interpolation do
        rule %r/\}\}/, Str::Interpol, :pop!
        mixin :root
      end
    end
  end
end
