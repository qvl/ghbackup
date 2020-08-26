package ghbackup

import (
	"reflect"
	"testing"
)

func Test_maskSecrets(t *testing.T) {
	type args struct {
		values  []string
		secrets []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "identicall",
			args: args{
				values:  []string{"ok", "haha secrethaha", "sdjdsajsasecretsdsasecret,tmp"},
				secrets: []string{"ok", "haha secrethaha", "sdjdsajsasecretsdsasecret,tmp"},
			},
			want: []string{"###", "###", "###"},
		},
		{
			name: "generic",
			args: args{
				values:  []string{"ok", "haha secrethaha", "sdjdsajsasecretsdsasecret,tmp"},
				secrets: []string{"secret"},
			},
			want: []string{"ok", "haha ###haha", "sdjdsajsa###sdsa###,tmp"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskSecrets(tt.args.values, tt.args.secrets); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("maskSecrets() = %v, want %v", got, tt.want)
			}
		})
	}
}
