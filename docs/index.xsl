<?xml version="1.0" encoding="UTF-8"?>

<xsl:stylesheet version="1.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<xsl:output
    method="html"
    encoding="utf-8"
    indent="yes"
    version="5.0"
    doctype-system="about:legacy-compat" />

<xsl:variable name="curr-lang">
    <xsl:value-of select="/docs/@lang" />
</xsl:variable>

<!-- 获取当前文档的语言名称，如果水存在，则直接采用 @lang 属性 -->
<xsl:variable name="curr-lang-title">
    <xsl:variable name="title">
        <xsl:value-of select="document('locales.xml')/locales/locale[@id=$curr-lang]/@title" />
    </xsl:variable>

    <xsl:choose>
        <xsl:when test="$title=''">
            <xsl:value-of select="$curr-lang" />
        </xsl:when>
        <xsl:otherwise>
            <xsl:value-of select="$title" />
        </xsl:otherwise>
    </xsl:choose>
</xsl:variable>

<xsl:template match="/">
    <html>
        <head>
            <title><xsl:value-of select="docs/title" /></title>
            <meta charset="UTF-8" />
            <meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1"/>
            <link rel="icon" type="image/png" href="./icon.png" />
            <link rel="canonical" href="{document('config.xml')/config/url}" />
            <link rel="stylesheet" type="text/css" href="./index.css" />
            <link rel="license" href="{/docs/liense/@url}" />
            <script src="./index.js"></script>
        </head>

        <body>
            <header>
                <div class="wrap">
                    <h1>
                        <img src="./icon.svg" />
                        <xsl:value-of select="document('config.xml')/config/name" />
                        <span class="version">&#160;(<xsl:value-of select="document('config.xml')/config/version" />)</span>
                    </h1>

                    <div class="menus" role="navigation">
                        <xsl:for-each select="docs/doc[not(@parent)]">
                            <a class="menu" href="#{@id}"><xsl:value-of select="@title" /></a>
                        </xsl:for-each>
                        <a class="menu" href="{document('config.xml')/config/repo}">Github</a>

                        <span class="drop-menus" role="menu">
                            <a class="menu">
                            <xsl:value-of select="$curr-lang-title" />
                            <span aria-hiddren="true">&#160;&#x25bc;</span>
                            </a>
                            <ul>
                            <xsl:for-each select="document('locales.xml')/locales/locale">
                                <li><a href="{@href}"><xsl:value-of select="@title" /></a></li>
                            </xsl:for-each>
                            </ul>
                        </span>
                    </div>
                </div>
            </header>

            <main>
                <xsl:for-each select="docs/doc[not(@parent)]">
                    <xsl:call-template name="article">
                        <xsl:with-param name="doc" select="." />
                    </xsl:call-template>
                </xsl:for-each>
            </main>

            <footer>
                <div class="wrap">
                <xsl:copy-of select="docs/footer/node()" />
                </div>
                <a href="#" class="goto-top"></a>
            </footer>
        </body>
    </html>
</xsl:template>

<xsl:template name="article">
    <xsl:param name="doc" />

    <article id="{$doc/@id}">
        <xsl:choose> <!-- 根据是否存在 $parent，决定是用 h3 还是 h2 标签 -->
            <xsl:when test="$doc/@parent">
                <h3>
                    <xsl:value-of select="$doc/@title" />
                    <a class="link" href="#{$doc/@id}">&#160;&#160;&#128279;</a>
                </h3>
            </xsl:when>
            <xsl:otherwise>
                <h2>
                    <xsl:value-of select="$doc/@title" />
                    <a class="link" href="#{$doc/@id}">&#160;&#160;&#128279;</a>
                </h2>
            </xsl:otherwise>
        </xsl:choose>

        <xsl:copy-of select="$doc/node()" />

        <xsl:for-each select="/docs/doc[@parent=$doc/@id]">
            <xsl:call-template name="article">
                <xsl:with-param name="doc" select="." />
            </xsl:call-template>
        </xsl:for-each>

        <xsl:for-each select="document('types.xml')/types/types[@parent=$doc/@id]/type">
            <xsl:call-template name="type">
                <xsl:with-param name="type" select="." />
                <xsl:with-param name="parent" select="$doc/@id" />
            </xsl:call-template>
        </xsl:for-each>
    </article>
</xsl:template>

<xsl:variable name="header-locale" select="/docs/type-locale/header" />
<xsl:variable name="doc-types" select="/docs/types" />

<!-- 将类型显示为一个 table -->
<xsl:template name="type">
    <xsl:param name="type" />
    <xsl:param name="parent" />

    <xsl:variable name="type-locale" select="$doc-types[@parent=$parent]/type[@name=$type/@name]" />

    <article id="type_{$type/@name}">
        <h3>
            <xsl:value-of select="$type/@name" />
            <a class="link" href="#type_{$type/@name}">&#160;&#160;&#128279;</a>
        </h3>
        <xsl:copy-of select="$type-locale/description/node()" />
        <xsl:if test="item">
            <table>
                <thead>
                    <tr>
                        <th><xsl:copy-of select="$header-locale/name" /></th>
                        <th><xsl:copy-of select="$header-locale/type" /></th>
                        <th><xsl:copy-of select="$header-locale/required" /></th>
                        <th><xsl:copy-of select="$header-locale/description" /></th>
                    </tr>
                </thead>

                <tbody>
                    <xsl:for-each select="$type/item">
                    <xsl:variable name="name" select="@name" />
                    <tr>
                        <th><xsl:value-of select="@name" /></th>
                        <td><xsl:value-of select="@type" /></td>
                        <td>
                            <xsl:call-template name="checkbox">
                                <xsl:with-param name="chk" select="@required" />
                            </xsl:call-template>
                        </td>
                        <td><xsl:copy-of select="$type-locale/item[@name=$name]/node()" /></td>
                    </tr>
                    </xsl:for-each>
                </tbody>
            </table>
        </xsl:if>
    </article>
</xsl:template>

<xsl:template name="checkbox">
    <xsl:param name="chk" />
    <xsl:choose>
        <xsl:when test="$chk='true'">
            <input type="checkbox" checked="true" disabled="true" />
        </xsl:when>
        <xsl:otherwise>
            <input type="checkbox" disabled="true" />
        </xsl:otherwise>
    </xsl:choose>
</xsl:template>

</xsl:stylesheet>
