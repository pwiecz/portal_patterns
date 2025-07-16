#version 150

uniform mat4 Matrix;
in vec2 Position;
in vec2 UV;

out vec2 Frag_UV;

void main()
{
	vec4 pos = vec4(Position, 0.0, 1.0);
	gl_Position = Matrix * pos;
	Frag_UV = UV;
}