_pj_projects() {
    local projects=(${(f)"$(pj list -n 2>/dev/null)"})
    compadd -S '' -- $projects
}

_pj_tags() {
    local tags=(${(f)"$(pj list -o tags 2>/dev/null)"})
    compadd -S '' -- $tags
}

_pj_filter_fields() {
    compadd -S '' -- name= path= editor= desc= tag=
}

_pj() {
    local -a commands=(
        'a:Add a project to the catalog'
        'add:Add a project to the catalog'
        'ls:List projects in the catalog'
        'list:List projects in the catalog'
        'rm:Remove a project from the catalog'
        'o:Open project in editor'
        'open:Open project in editor'
        'e:Edit project metadata'
        'edit:Edit project metadata'
        'create:Create a new project'
        'new:Create a new project'
        'show:Show project details'
        'cd:Change directory to project'
        'tag:Manage project tags'
        'init:Generate shell integration'
        'completion:Generate shell completions'
    )

    _arguments -C \
        '(-h --help)'{-h,--help}'[Show help]' \
        '(-c --catalog)'{-c,--catalog}'[Path to catalog file]:path:_files' \
        '1:command:->cmds' \
        '*::arg:->args'

    case $state in
        cmds) _describe 'command' commands ;;
        args)
            case $line[1] in
                a|add)
                    _arguments \
                        '*'{-t,--tag}'[Tags]:tag:_pj_tags' \
                        '(-n --name)'{-n,--name}'[Project name]:name:' \
                        '1:path:_files -/'
                    ;;
                ls|list)
                    _arguments \
                        '*'{-t,--tag}'[Tags]:tag:_pj_tags' \
                        '*'{-f,--filter}'[Filter projects]:filter:_pj_filter_fields' \
                        '(-o --output)'{-o,--output}'[Output format]:format:(table names paths json tags)'
                    ;;
                rm)
                    _arguments '1:project:_pj_projects'
                    ;;
                o|open)
                    _arguments '1:project:_pj_projects'
                    ;;
                e|edit)
                    _arguments \
                        '*'{-t,--tag}'[Tags]:tag:_pj_tags' \
                        '(--clear-tags)*--untag[Remove tags]:tag:_pj_tags' \
                        '(--untag)--clear-tags[Remove all tags]' \
                        '--desc[Set description]:description:' \
                        '--editor[Set editor]:editor:' \
                        '1:project:_pj_projects'
                    ;;
                create|new)
                    _arguments \
                        '*'{-t,--tag}'[Tags]:tag:_pj_tags' \
                        '--at[Parent directory]:directory:_files -/' \
                        '--desc[Project description]:description:' \
                        '--editor[Editor command]:editor:' \
                        '--no-git[Skip git initialization]' \
                        '--adopt[Adopt an existing directory]' \
                        '--no-input[Never prompt]' \
                        '1:name:'
                    ;;
                show)
                    _arguments \
                        '--path[Output only the path]' \
                        '1:project:_pj_projects'
                    ;;
                cd)
                    _arguments '1:project:_pj_projects'
                    ;;
                tag)
                    if (( CURRENT == 2 )); then
                        local -a tag_commands=('detect:Detect language tags from project files')
                        _describe 'tag command' tag_commands
                    else
                        _arguments \
                            '*'{-t,--tag}'[Match tags]:tag:_pj_tags' \
                            '*'{-f,--filter}'[Filter projects]:filter:_pj_filter_fields' \
                            '--apply[Save detected tags]'
                    fi
                    ;;
                completion)
                    _arguments '1:shell:(zsh)'
                    ;;
            esac
            ;;
    esac
}

compdef _pj pj
